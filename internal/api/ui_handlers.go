//go:build ui

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/danielgtaylor/huma/v2"
)

// isHTMXRequest checks if the request is coming from HTMX
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// isHTMXBoost checks if this is a boosted navigation request
func isHTMXBoost(r *http.Request) bool {
	return r.Header.Get("HX-Boosted") == "true"
}

// uiRenderExternalIDPDisabled renders the page shown when external IDP is enabled.
func (s *Server) uiRenderExternalIDPDisabled(w http.ResponseWriter, r *http.Request) {
	s.renderWithUser(w, r, "external_idp_disabled.gohtml", nil)
}

// uiRenderHome renders the home page with a paginated list of articles.
func (s *Server) uiRenderHome(w http.ResponseWriter, r *http.Request) {
	input := &ArticlePaginationInput{
		Page:  1,
		Limit: 20,
	}

	pageStr := r.URL.Query().Get("page")
	p, err := strconv.Atoi(pageStr)
	if err == nil && p > 0 {
		input.Page = p
	}

	resp, err := s.handleGetArticles(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	s.renderWithUser(w, r, "home.gohtml", resp.Body)
}

// uiRenderArticle renders a single article page.
func (s *Server) uiRenderArticle(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	input := &ArticleSlugInput{Slug: slug}

	resp, err := s.handleGetArticleJSON(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	wikiContent, err := s.getRenderedHTML(r.Context(), resp.Body.PublicArticle)
	if err != nil {
		s.uiError(w, r, fmt.Errorf("failed to render markdown: %w", err))
		return
	}

	if s.hasActivePlugins() {
		pluginCtx := map[string]any{
			"User": getUserFromContext(r.Context()),
			"Slug": slug,
		}

		finalBody, err := executePlugins(
			r.Context(),
			s.PluginManager,
			"onArticleRender",
			wikiContent,
			pluginCtx,
			s.db.CreateLogEntry,
		)
		if err != nil {
			s.uiError(w, r, fmt.Errorf("failed to execute plugins: %w", err))
			return
		}

		wikiContent = finalBody
	}

	resp.Body.PublicArticle.Data = wikiContent

	s.renderWithUser(w, r, "article.gohtml", resp.Body.PublicArticle)
}

// uiRenderHistory renders the history page for an article.
func (s *Server) uiRenderHistory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	input := &ArticleSlugInput{Slug: slug}

	resp, err := s.handleGetArticleHistory(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	data := struct {
		Slug    string
		History []*models.History
	}{
		Slug:    slug,
		History: resp.Body.History,
	}

	s.renderWithUser(w, r, "history.gohtml", data)
}

// uiRenderPastVersion renders a specific past version of an article.
func (s *Server) uiRenderPastVersion(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	verStr := r.PathValue("version")
	version, _ := strconv.Atoi(verStr)

	article, err := s.db.GetArticleBySlug(r.Context(), slug)
	if err != nil || article == nil {
		s.uiError(w, r, huma.Error404NotFound("Article not found"))
		return
	}

	content, err := s.db.GetArticleVersion(r.Context(), article.Id, version)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	var buf bytes.Buffer
	err = s.renderer.RenderHTML(r.Context(), &buf, content)
	if err != nil {
		s.uiError(w, r, fmt.Errorf("failed to render markdown: %w", err))
		return
	}

	viewData := &PublicArticle{
		Id:      article.Id,
		Title:   article.Title,
		Slug:    article.Slug,
		Version: version,
		Data:    buf.String(),
	}
	s.renderWithUser(w, r, "article.gohtml", viewData)
}

// uiRenderLogin renders the login page.
func (s *Server) uiRenderLogin(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{}

	if r.URL.Query().Get("error") == "1" {
		data["Error"] = "Invalid credentials"
	}

	s.renderWithUser(w, r, "login.gohtml", data)
}

// uiHandleLoginSubmit handles the submission of the login form.
func (s *Server) uiHandleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.uiError(w, r, huma.Error400BadRequest("Bad Request"))
		return
	}

	input := &LoginInput{}
	input.Body.Email = r.FormValue("email")
	input.Body.Password = r.FormValue("password")
	input.Body.OTP = r.FormValue("otp")

	resp, err := s.handleLogin(r.Context(), input)
	if err != nil {
		if isHTMXRequest(r) || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		s.renderWithUser(w, r, "login.gohtml", map[string]string{"Error": "Invalid credentials"})
		return
	}

	for _, cookieStr := range resp.Cookies {
		w.Header().Add("Set-Cookie", cookieStr)
	}

	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", "/dashboard")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Login successful"))
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// uiHandleLogout handles user logout.
func (s *Server) uiHandleLogout(w http.ResponseWriter, r *http.Request) {
	resp, _ := s.handleLogout(r.Context(), nil)
	for _, cookieStr := range resp.Cookies {
		w.Header().Add("Set-Cookie", cookieStr)
	}

	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// uiRenderNewArticle displays the form to name a new article.
func (s *Server) uiRenderNewArticle(w http.ResponseWriter, r *http.Request) {
	s.renderWithUser(w, r, "new_article.gohtml", nil)
}

// uiActionCreateIntent handles the intent to create a new article.
func (s *Server) uiActionCreateIntent(w http.ResponseWriter, r *http.Request) {
	input := &CreateArticleInput{}
	input.Body.Title = strings.TrimSpace(r.FormValue("title"))

	if input.Body.Title == "" {
		s.renderWithUser(w, r, "new_article.gohtml", map[string]string{
			"Error": "Title is required",
		})
		return
	}

	resp, err := s.handleCreateArticle(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	redirectUrl := fmt.Sprintf("/editor/%d", resp.Body.DraftID)

	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", redirectUrl)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

// uiActionEditIntent handles the intent to edit an existing article.
func (s *Server) uiActionEditIntent(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	input := &ArticleSlugForDraftInput{Slug: slug}

	resp, err := s.handleCreateDraft(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	redirectUrl := fmt.Sprintf("/editor/%d", resp.Body.Draft.Id)
	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

// uiRenderEditor renders the article editor page.
func (s *Server) uiRenderEditor(w http.ResponseWriter, r *http.Request) {
	draftID, _ := strconv.Atoi(r.PathValue("draftID"))
	input := &DraftIDInput{ID: draftID}

	resp, err := s.handleGetDraft(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	s.renderWithUser(w, r, "editor.gohtml", resp.Body.Draft)
}

// uiActionSaveDraft handles saving a draft of an article.
func (s *Server) uiActionSaveDraft(w http.ResponseWriter, r *http.Request) {
	draftID, _ := strconv.Atoi(r.PathValue("draftID"))

	err := r.ParseForm()
	if err != nil {
		s.uiError(w, r, huma.Error400BadRequest("Bad form data"))
		return
	}

	input := &UpdateDraftInput{ID: draftID}
	input.Body.Content = r.FormValue("content")

	_, err = s.handleUpdateDraft(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/editor/%d", draftID), http.StatusFound)
}

// uiActionPublishDraft handles publishing a draft of an article.
func (s *Server) uiActionPublishDraft(w http.ResponseWriter, r *http.Request) {
	draftID, _ := strconv.Atoi(r.PathValue("draftID"))

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		s.uiError(w, r, fmt.Errorf("bad form data: %w", err))
		return
	}
	content := r.FormValue("content")

	updateInput := &UpdateDraftInput{ID: draftID}
	updateInput.Body.Content = content
	_, err = s.handleUpdateDraft(r.Context(), updateInput)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	draftResp, err := s.handleGetDraft(r.Context(), &DraftIDInput{ID: draftID})
	if err != nil {
		s.uiError(w, r, err)
		return
	}
	slug := draftResp.Body.Draft.ArticleSlug

	_, err = s.handlePublishDraft(r.Context(), &DraftIDInput{ID: draftID})
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/wiki/%s", slug), http.StatusFound)
}

// uiActionDiscardDraft handles discarding a draft of an article.
func (s *Server) uiActionDiscardDraft(w http.ResponseWriter, r *http.Request) {
	draftID, _ := strconv.Atoi(r.PathValue("draftID"))

	_, err := s.handleDiscardDraft(r.Context(), &DraftIDInput{ID: draftID})
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// uiRenderDashboard renders the user's dashboard.
func (s *Server) uiRenderDashboard(w http.ResponseWriter, r *http.Request) {
	draftsResp, err := s.handleGetMyDrafts(r.Context(), nil)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	articlesResp, _ := s.handleGetArticlesByUser(r.Context(), &ArticleListInput{})

	data := struct {
		Drafts   []*PublicDraft
		Articles []*PublicArticle
	}{
		Drafts:   draftsResp.Body.Drafts,
		Articles: nil,
	}
	if articlesResp != nil {
		data.Articles = articlesResp.Body.Articles
	}

	s.renderWithUser(w, r, "dashboard.gohtml", data)
}

// uiRenderOrphans renders the page for orphaned articles.
func (s *Server) uiRenderOrphans(w http.ResponseWriter, r *http.Request) {
	resp, err := s.handleGetOrphans(r.Context(), nil)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	s.renderWithUser(
		w,
		r,
		"orphans.gohtml",
		struct{ Articles []*PublicArticle }{Articles: resp.Body.Articles},
	)
}

// uiActionDeleteArticle handles deleting an article.
func (s *Server) uiActionDeleteArticle(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	input := &ArticleSlugInput{Slug: slug}

	_, err := s.handleDeleteArticle(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// uiRenderLogs renders the logs page.
func (s *Server) uiRenderLogs(w http.ResponseWriter, r *http.Request) {
	input := &LogsPaginationInput{
		Page:  1,
		Limit: 50,
	}

	pageStr := r.URL.Query().Get("page")
	p, err := strconv.Atoi(pageStr)
	if err == nil && p > 0 {
		input.Page = p
	}

	resp, err := s.handleGetLogs(r.Context(), input)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	s.renderWithUser(w, r, "logs.gohtml", resp.Body)
}

// uiError logs the error to the database and renders a user-friendly error page.
func (s *Server) uiError(w http.ResponseWriter, r *http.Request, err error) {
	userEmail := "Anonymous"
	if user := getUserFromContext(r.Context()); user != nil {
		userEmail = user.Email
	}

	statusCode := http.StatusInternalServerError
	statusText := "Internal Server Error"
	message := "Something went wrong on our end. The error has been logged for review."

	var statusErr huma.StatusError
	if errors.As(err, &statusErr) {
		statusCode = statusErr.GetStatus()
		statusText = http.StatusText(statusCode)
		message = statusErr.Error()
	}

	logLevel := models.LevelError
	if statusCode < 500 {
		logLevel = models.LevelWarning
	}

	go func() {
		_ = s.db.CreateLogEntry(
			context.Background(),
			logLevel,
			"UI",
			fmt.Sprintf("Handler Error [%d]: %v", statusCode, err),
			fmt.Sprintf("Path: %s | User: %s", r.URL.Path, userEmail),
		)
	}()

	w.WriteHeader(statusCode)

	data := struct {
		StatusCode int
		StatusText string
		Message    string
	}{
		StatusCode: statusCode,
		StatusText: statusText,
		Message:    message,
	}

	s.renderWithUser(w, r, "error.gohtml", data)
}

// render executes a named template into a buffer before writing to the response.
func (s *Server) render(w http.ResponseWriter, r *http.Request, tmplName string, pageData any) {
	if s.compiledTemplates == nil {
		http.Error(w, "Templates not initialized. Call app.InitTemplates()", 500)
		return
	}

	tmpl, ok := s.compiledTemplates[tmplName]
	if !ok {
		if !strings.HasSuffix(tmplName, ".gohtml") {
			tmpl, ok = s.compiledTemplates[tmplName+".gohtml"]
		}
	}

	if !ok {
		fmt.Printf("Template Not Found: %s. Available: %v\n", tmplName, s.getAvailableTemplates())
		http.Error(w, "Template not found", 500)
		return
	}

	var buf bytes.Buffer

	if isHTMXRequest(r) && isHTMXBoost(r) {
		err := tmpl.ExecuteTemplate(&buf, "content", pageData)
		if err != nil {
			fmt.Printf("Template Error [%s]: %v\n", tmplName, err)
			http.Error(w, "Template rendering failed", 500)
			return
		}
	} else {
		err := tmpl.ExecuteTemplate(&buf, "base.gohtml", pageData)
		if err != nil {
			fmt.Printf("Template Error [%s]: %v\n", tmplName, err)
			http.Error(w, "Template rendering failed", 500)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if isHTMXRequest(r) {
		w.Header().Set("HX-Trigger", "contentUpdated")
		w.Header().Set("HX-Retarget", "main")
	}

	_, _ = buf.WriteTo(w)
}

// getAvailableTemplates returns a list of available templates.
func (s *Server) getAvailableTemplates() []string {
	keys := make([]string, 0, len(s.compiledTemplates))
	for k := range s.compiledTemplates {
		keys = append(keys, k)
	}
	return keys
}

// renderWithUser wraps data with User context.
func (s *Server) renderWithUser(w http.ResponseWriter, r *http.Request, tmplName string, data any) {
	user := getUserFromContext(r.Context())

	payload := templateData{
		User:     user,
		Data:     data,
		WikiName: s.WikiName,
	}
	s.render(w, r, tmplName, payload)
}

func (s *Server) uiRenderUser(w http.ResponseWriter, r *http.Request) {
	s.renderWithUser(w, r, "user.gohtml", nil)
}

func (s *Server) uiActionUpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if newPassword != confirmPassword {
		s.renderWithUser(
			w,
			r,
			"user.gohtml",
			map[string]string{"Error": "New passwords do not match"},
		)
		return
	}

	dbUser, err := s.db.GetUserByID(r.Context(), user.Id)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	if dbUser == nil {
		s.uiError(w, r, huma.Error404NotFound("User not found"))
		return
	}

	if !utils.CheckPassword(currentPassword, dbUser.Hash) {
		s.renderWithUser(
			w,
			r,
			"user.gohtml",
			map[string]string{"Error": "Incorrect current password"},
		)
		return
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	dbUser.Hash = hashedPassword
	err = s.db.UpdateUser(r.Context(), dbUser, "hash")
	if err != nil {
		s.uiError(w, r, err)
		return
	}

	s.renderWithUser(
		w,
		r,
		"user.gohtml",
		map[string]string{"Success": "Password updated successfully"},
	)
}

// uiRenderOTPSettings renders the OTP settings page.
func (s *Server) uiRenderOTPSettings(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{}

	if r.URL.Query().Get("success") == "1" {
		data["Success"] = "Two-factor authentication enabled successfully"
	}

	s.renderWithUser(w, r, "otp_settings.gohtml", data)
}

// uiHandleOTPStartEnrollment handles starting the OTP enrollment process.
func (s *Server) uiHandleOTPStartEnrollment(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.uiError(w, r, huma.Error400BadRequest("Bad form data"))
		return
	}

	password := r.FormValue("password")
	if password == "" {
		s.renderWithUser(
			w,
			r,
			"otp_settings.gohtml",
			map[string]string{"Error": "Password is required"},
		)
		return
	}

	input := &OTPStartEnrollmentInput{}
	input.Body.Password = password

	resp, err := s.handleStartOTPEnrollment(r.Context(), input)
	if err != nil {
		s.renderWithUser(
			w,
			r,
			"otp_settings.gohtml",
			map[string]string{"Error": "Invalid password"},
		)
		return
	}

	data := struct {
		QRCode      template.URL
		Secret      string
		BackupCodes []string
	}{
		QRCode:      template.URL(resp.Body.QRCode),
		Secret:      resp.Body.Code,
		BackupCodes: resp.Body.BackupCodes,
	}

	s.renderWithUser(w, r, "otp_enroll.gohtml", data)
}

// uiRenderOTPEnroll renders the OTP enrollment page.
func (s *Server) uiRenderOTPEnroll(w http.ResponseWriter, r *http.Request) {
	s.renderWithUser(w, r, "otp_enroll.gohtml", nil)
}

// uiHandleOTPVerify handles OTP verification during enrollment.
func (s *Server) uiHandleOTPVerify(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		s.uiError(w, r, huma.Error400BadRequest("Bad form data"))
		return
	}

	code := r.FormValue("code")
	if code == "" {
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Verification code is required"))
			return
		}
		s.renderWithUser(
			w,
			r,
			"otp_enroll.gohtml",
			map[string]string{"Error": "Verification code is required"},
		)
		return
	}

	input := &OTPCompleteEnrollmentInput{}
	input.Code = code

	_, err = s.handleCompleteOTPEnrollment(r.Context(), input)
	if err != nil {
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid verification code"))
			return
		}
		s.renderWithUser(
			w,
			r,
			"otp_enroll.gohtml",
			map[string]string{"Error": "Invalid verification code"},
		)
		return
	}

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Two-factor authentication enabled successfully"))
		return
	}

	s.renderWithUser(
		w,
		r,
		"otp_settings.gohtml",
		map[string]string{"Success": "Two-factor authentication enabled successfully"},
	)
}

// uiHandleOTPDisable handles disabling OTP.
func (s *Server) uiHandleOTPDisable(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	input := &OTPRemoveInput{}

	_, err := s.handleRemoveOTP(r.Context(), input)
	if err != nil {
		s.renderWithUser(
			w,
			r,
			"otp_settings.gohtml",
			map[string]string{"Error": "Failed to disable two-factor authentication"},
		)
		return
	}

	s.renderWithUser(
		w,
		r,
		"otp_settings.gohtml",
		map[string]string{"Success": "Two-factor authentication disabled successfully"},
	)
}
