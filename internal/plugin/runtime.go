//go:build plugins

package plugin

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"wikilite/pkg/models"

	"github.com/jellydator/ttlcache/v3"
	"github.com/microcosm-cc/bluemonday"
	"modernc.org/quickjs"
)

const (
	cacheTtl  = 30 * time.Minute
	cacheSize = 1000
)

// Manager manages a set of fixed workers that own QuickJS VMs.
type Manager struct {
	Store Store

	sanitizer *bluemonday.Policy

	jobQueue chan jobRequest
	stopChan chan struct{}

	cache      *ttlcache.Cache[string, string]
	jsPkgsPath string

	Plugins   []Plugin
	pluginIDs []string
	wg        sync.WaitGroup
}

// jobType distinguishes between pipeline hooks and direct actions.
type jobType int

const (
	jobTypePipeline jobType = iota
	jobTypeAction
)

// jobRequest represents a task for a worker.
type jobRequest struct {
	respChan chan jobResponse

	contextJSON string

	hook  string
	input string

	pluginID string
	action   string
	payload  string

	kind jobType
}

// jobResponse carries the result back to the caller.
type jobResponse struct {
	err    error
	result string
	errors []Error
}

// NewManager creates a new plugin manager with a fixed worker pool.
func NewManager(dbPath string, pluginDir string, jsPkgsPath string) (*Manager, error) {
	store, err := newBoltStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	plugins, err := loadFromDirectory(pluginDir)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	pluginIDs := make([]string, len(plugins))
	for i, p := range plugins {
		pluginIDs[i] = p.ID
	}

	workerCount := max(runtime.NumCPU(), 4)

	cache := ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](cacheTtl),
		ttlcache.WithCapacity[string, string](cacheSize),
	)
	go cache.Start()

	m := &Manager{
		Store:      store,
		Plugins:    plugins,
		pluginIDs:  pluginIDs,
		jsPkgsPath: jsPkgsPath,
		jobQueue:   make(chan jobRequest, workerCount*10),
		stopChan:   make(chan struct{}),
		sanitizer:  bluemonday.UGCPolicy(),
		cache:      cache,
	}

	for i := 0; i < workerCount; i++ {
		m.wg.Add(1)
		go m.workerLoop(i)
	}

	return m, nil
}

// Close shuts down the manager and all workers.
func (m *Manager) Close() error {
	if m.cache != nil {
		m.cache.Stop()
	}

	close(m.stopChan)
	m.wg.Wait()

	if m.Store != nil {
		return m.Store.Close()
	}
	return nil
}

// HasPlugins returns true if the manager has any plugins loaded.
func (m *Manager) HasPlugins() bool {
	return len(m.Plugins) > 0
}

// ExecutePipeline checks the cache before sending a job to the worker pool.
func (m *Manager) ExecutePipeline(
	hookName string,
	initialInput string,
	contextData map[string]any,
) (string, []Error, error) {
	var slug string
	var role int

	if contextData != nil {
		s, ok := contextData["Slug"].(string)
		if ok {
			slug = s
		}

		u, ok := contextData["User"].(*models.User)
		if ok && u != nil {
			role = int(u.Role)
		}
	}

	var cacheKey string
	if slug != "" && hookName == "onArticleRender" {
		hash := md5.Sum([]byte(initialInput))
		cacheKey = fmt.Sprintf(
			"pipeline:%s:%s:%s:%d",
			hookName,
			slug,
			hex.EncodeToString(hash[:]),
			role,
		)

		if item := m.cache.Get(cacheKey); item != nil {
			return item.Value(), nil, nil
		}
	}

	ctxJSON := ""
	if contextData != nil {
		b, _ := json.Marshal(contextData)
		ctxJSON = string(b)
	}

	respChan := make(chan jobResponse, 1)
	m.jobQueue <- jobRequest{
		kind:        jobTypePipeline,
		hook:        hookName,
		input:       initialInput,
		contextJSON: ctxJSON,
		respChan:    respChan,
	}

	resp := <-respChan

	if resp.err == nil && len(resp.errors) == 0 && cacheKey != "" {
		m.cache.Set(cacheKey, resp.result, ttlcache.DefaultTTL)
	}

	return resp.result, resp.errors, resp.err
}

// ExecutePluginAction sends an action job to the worker pool and invalidates cache on success.
func (m *Manager) ExecutePluginAction(
	pluginID string,
	action string,
	payload string,
	contextData map[string]any,
) (string, error) {
	ctxJSON := ""
	if contextData != nil {
		b, _ := json.Marshal(contextData)
		ctxJSON = string(b)
	}

	respChan := make(chan jobResponse, 1)
	m.jobQueue <- jobRequest{
		kind:        jobTypeAction,
		pluginID:    pluginID,
		action:      action,
		payload:     payload,
		contextJSON: ctxJSON,
		respChan:    respChan,
	}

	resp := <-respChan

	if resp.err == nil && contextData != nil {
		slug, ok := contextData["Slug"].(string)
		if ok && slug != "" {
			target := ":" + slug + ":"
			keysToDelete := []string{}

			for k := range m.cache.Items() {
				if strings.Contains(k, target) {
					keysToDelete = append(keysToDelete, k)
				}
			}

			for _, k := range keysToDelete {
				m.cache.Delete(k)
			}
		}
	}

	return resp.result, resp.err
}

// workerLoop is the main loop for a single thread-bound worker.
func (m *Manager) workerLoop(id int) {
	defer m.wg.Done()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	vm, err := quickjs.NewVM()
	if err != nil {
		fmt.Printf("Worker %d failed to initialize VM: %v\n", id, err)
		return
	}
	defer func(vm *quickjs.VM) {
		vErr := vm.Close()
		if vErr != nil {
			log.Printf("Worker %d failed to close VM: %v\n", id, vErr)
		}
	}(vm)

	err = m.initVM(vm)
	if err != nil {
		fmt.Printf("Worker %d failed to load environment: %v\n", id, err)
		return
	}

	for {
		select {
		case <-m.stopChan:
			return
		case job := <-m.jobQueue:
			m.processJob(vm, job)
		}
	}
}

// processJob handles a single request using the worker's VM.
func (m *Manager) processJob(vm *quickjs.VM, job jobRequest) {
	var resp jobResponse

	var ctxArg any
	if job.contextJSON != "" {
		v, err := vm.CallValue("JSON.parse", job.contextJSON)
		if err == nil {
			ctxArg = v
			defer v.Free()
		}
	}

	switch job.kind {
	case jobTypePipeline:
		resp = m.processPipelineJob(vm, job, ctxArg)
	default:
		resp = m.processActionJob(vm, job, ctxArg)
	}

	job.respChan <- resp
}

// processPipelineJob handles pipeline-type jobs.
func (m *Manager) processPipelineJob(
	vm *quickjs.VM,
	job jobRequest,
	ctxArg any,
) jobResponse {
	var resp jobResponse

	res, err := vm.Call("__run_pipeline_json", job.hook, job.input, ctxArg)
	if err != nil {
		resp.err = err
		return resp
	}

	resStr, ok := res.(string)
	if !ok {
		resp.err = fmt.Errorf("unexpected pipeline result type")
		return resp
	}

	var pipelineResult struct {
		Content string `json:"content"`
		Errors  []struct {
			PluginID string `json:"pluginId"`
			Hook     string `json:"hook"`
			Error    string `json:"error"`
		} `json:"errors"`
	}

	err = json.Unmarshal([]byte(resStr), &pipelineResult)
	if err != nil {
		resp.err = fmt.Errorf("failed to parse pipeline result: %w", err)
		return resp
	}

	resp.result = pipelineResult.Content
	for _, e := range pipelineResult.Errors {
		resp.errors = append(resp.errors, Error{
			PluginID: e.PluginID,
			Hook:     e.Hook,
			Error:    e.Error,
		})
	}

	return resp
}

// processActionJob handles action-type jobs.
func (m *Manager) processActionJob(vm *quickjs.VM, job jobRequest, ctxArg any) jobResponse {
	var resp jobResponse

	res, err := vm.Call("__run_plugin_action_json", job.pluginID, job.action, job.payload, ctxArg)
	if err != nil {
		resp.err = err
		return resp
	}

	resStr, ok := res.(string)
	if !ok {
		resp.err = fmt.Errorf("unexpected action result type")
		return resp
	}

	var actionResult struct {
		Error string `json:"error"`
	}

	if parseErr := json.Unmarshal([]byte(resStr), &actionResult); parseErr == nil &&
		actionResult.Error != "" {
		resp.err = fmt.Errorf("plugin action error: %s", actionResult.Error)
		return resp
	}

	resp.result = resStr
	return resp
}

// initVM sets up the JS environment (libs, plugins, shims) for a VM.
func (m *Manager) initVM(vm *quickjs.VM) error {
	err := m.injectHostAPI(vm)
	if err != nil {
		return err
	}

	var libs string
	if m.jsPkgsPath != "" {
		content, err := os.ReadFile(m.jsPkgsPath)
		if err != nil {
			return fmt.Errorf("failed to load custom jspkgs: %w", err)
		}
		libs = string(content)
	} else {
		libs = jsLibraries
	}

	_, err = vm.Eval(libs, quickjs.EvalGlobal)
	if err != nil {
		return fmt.Errorf("js libraries error: %w", err)
	}

	for _, p := range m.Plugins {
		safeID := fmt.Sprintf("PLUGIN_%s", p.ID)
		wrapper := fmt.Sprintf(`
			globalThis['%[1]s'] = (function() {
				// --- User Code Start ---
				%[2]s
				// --- User Code End ---
				var exports = {};
				if (typeof onArticleRender === 'function') exports.onArticleRender = onArticleRender;
				if (typeof onAction === 'function') exports.onAction = onAction;
				return exports;
			})();
		`, safeID, p.Script)

		_, err = vm.Eval(wrapper, quickjs.EvalGlobal)
		if err != nil {
			return fmt.Errorf("plugin %s error: %w", p.ID, err)
		}
	}

	return m.injectPipelineExecutor(vm)
}

// injectHostAPI creates a Host object in JS that allows plugins to store data.
func (m *Manager) injectHostAPI(vm *quickjs.VM) error {
	err := vm.RegisterFunc("__internal_sanitize_html", func(dirty string) string {
		return m.sanitizer.Sanitize(dirty)
	}, false)
	if err != nil {
		return err
	}

	err = vm.RegisterFunc("__internal_storage_get", func(pluginID string, key string) string {
		val, _ := m.Store.Get(pluginID, key)
		return val
	}, false)
	if err != nil {
		return err
	}

	err = vm.RegisterFunc(
		"__internal_storage_set",
		func(pluginID string, key string, val string) int32 {
			err := m.Store.Set(pluginID, key, val)
			if err == nil {
				return 1
			}
			return 0
		},
		false,
	)
	if err != nil {
		return err
	}

	err = vm.RegisterFunc(
		"__internal_storage_delete",
		func(pluginID string, key string) int32 {
			err := m.Store.Delete(pluginID, key)
			if err == nil {
				return 1
			}
			return 0
		},
		false,
	)
	if err != nil {
		return err
	}

	err = vm.RegisterFunc(
		"__internal_storage_list",
		func(pluginID string, prefix string) string {
			keys, _ := m.Store.List(pluginID, prefix)
			data, _ := json.Marshal(keys)
			return string(data)
		},
		false,
	)
	if err != nil {
		return err
	}

	consoleLogShim := `
		globalThis.console = {
		  log: function (...args) {
			const msg = args.map((arg) => {
			  if (typeof arg === "object") {
				try { return JSON.stringify(arg); } catch (e) { return String(arg); }
			  }
			  return String(arg);
			}).join(" ");
			if (typeof __console_log === "function") __console_log(msg);
		  },
		  error: function (...args) { this.log("ERROR:", ...args); },
		  warn: function (...args) { this.log("WARN:", ...args); },
		  info: function (...args) { this.log("INFO:", ...args); },
		};
	`
	_, err = vm.Eval(consoleLogShim, quickjs.EvalGlobal)
	if err != nil {
		return err
	}

	err = vm.RegisterFunc("__console_log", func(msg string) {
		fmt.Println("[PLUGIN]", msg)
	}, false)
	if err != nil {
		return err
	}

	shim := `
		var Host = {
			storage: {
				get: function(k) { 
					var id = globalThis.__CURRENT_PLUGIN_ID;
					return __internal_storage_get(id, k); 
				},
				set: function(k, v) { 
					var id = globalThis.__CURRENT_PLUGIN_ID;
					return __internal_storage_set(id, k, v); 
				},
				delete: function(k) { 
					var id = globalThis.__CURRENT_PLUGIN_ID;
					return __internal_storage_delete(id, k); 
				},
				list: function(p) { 
					var id = globalThis.__CURRENT_PLUGIN_ID;
					var res = __internal_storage_list(id, p);
					try { 
						var parsed = JSON.parse(res); 
						return parsed || [];
					} catch(e) { return []; }
				},
				sanitize: function(html) {
					return __internal_sanitize_html(html);
				}
			}
		};
	`
	_, err = vm.Eval(shim, quickjs.EvalGlobal)
	return err
}

// injectPipelineExecutor creates a batched pipeline runner in JS.
func (m *Manager) injectPipelineExecutor(vm *quickjs.VM) error {
	var pluginIDsJS strings.Builder
	pluginIDsJS.WriteString("[")
	for i, id := range m.pluginIDs {
		if i > 0 {
			pluginIDsJS.WriteString(", ")
		}
		pluginIDsJS.WriteString("'")
		pluginIDsJS.WriteString(id)
		pluginIDsJS.WriteString("'")
	}
	pluginIDsJS.WriteString("]")

	pipelineJS := fmt.Sprintf(`
		function __run_pipeline(hook, content, ctx) {
			var plugins = %s;
			var current = content;
			var errors = [];
			
			for (var i = 0; i < plugins.length; i++) {
				var pid = plugins[i];
				globalThis.__CURRENT_PLUGIN_ID = pid;
				
				var p = globalThis['PLUGIN_' + pid];
				if (p && typeof p[hook] === 'function') {
					try {
						var res = p[hook](current, ctx);
						if (typeof res === 'string') {
							current = res;
						}
					} catch (e) {
						errors.push({
							pluginId: pid,
							hook: hook,
							error: String(e.message || e)
						});
					}
				}
			}
			
			return {
				content: current,
				errors: errors
			};
		}

		function __run_pipeline_json(hook, content, ctx) {
			var result = __run_pipeline(hook, content, ctx);
			return JSON.stringify(result);
		}

		function __run_plugin_action(pluginId, action, payloadStr, ctx) {
			globalThis.__CURRENT_PLUGIN_ID = pluginId;
			var p = globalThis['PLUGIN_' + pluginId];
			if (p && typeof p.onAction === 'function') {
				var payload = {};
				try { payload = JSON.parse(payloadStr); } catch(e) {}
				return p.onAction(action, payload, ctx);
			}
			return {error: "Plugin or action not found"};
		}

		function __run_plugin_action_json(pluginId, action, payloadStr, ctx) {
			var result = __run_plugin_action(pluginId, action, payloadStr, ctx);
			return JSON.stringify(result);
		}
	`, pluginIDsJS.String())

	_, err := vm.Eval(pipelineJS, quickjs.EvalGlobal)
	return err
}

// PipelineResult and PluginError structs can remain or be redefined if needed
type PipelineResult struct {
	Content string
	Errors  []Error
}

// Error represents an error that occurred in a plugin during execution.
type Error struct {
	PluginID string
	Hook     string
	Error    string
}
