# **Wikilite**

Wikilite is a lightweight, extensible wiki engine written in Go. It allows users to write documentation in Markdown with built-in version control, a drafting system, and role-based user management. It is designed to be API-first but includes an optional server-side rendered UI.

## **Features**

* **Markdown Support:** robust rendering with GFM extensions.
* **Drafting System:** Create, edit, and publish drafts without affecting the live article.
* **Version Control:** Automatic history tracking for every article.
* **User Management:** Role-based access (Read, Write, Admin) and external IDP support.
* **Orphan Detection:** Identify pages with no incoming links.
* **System Logging:** Integrated database logging for auditing.
* **Plugin Support:** Write custom JavaScript plugins to extend functionality.

## **Building**

Wikilite releases include three build configurations:

### **UI + Plugins (Recommended for Basic Deployments)**
Includes the HTML frontend and plugin support:

```
go build -tags "ui,plugins" -o wikilite cmd/main.go
```

### **Headless + Plugins**
API server with plugin support, no UI:

```
go build -tags plugins -o wikilite cmd/main.go
```

### **Headless**
Minimal API server, no UI or plugins:

```
go build -o wikilite cmd/main.go
```

## **Authentication/Authorization**
### **Local Auth** 
Wikilite supports the following authentication methods when using local auth:
* User/Password that sets a local JWT access token in an `httpOnly` cookie - ```/api/login```.
* User/Password that returns a local JWT access token for direct usage in API calls = ```/api/login/token```.

#### **Two-Factor Authentication (2FA)**
Users can enable TOTP-based two-factor authentication for enhanced security. Includes backup codes for account recovery and is compatible with most authenticator apps.

In the built-in UI:
* Navigate to `/user` to access account settings
* Enable 2FA and scan the QR code with your authenticator app
* Save backup codes for account recovery

### **External IdP Auth**
When using external IdP auth, Wikilite supports the following methods:
* JWT access token only if it includes an email address claim in ```Authorization: Bearer``` in the request header.
* JWT access token + an ID Token from the external IdP. 
  * Access token in ```Authorization: Bearer``` header.`
  * ID token in ```X-ID-Token``` header.

If an external user is not found in the system, they will be created automatically with `READ` access.

_Note_: If included in the build, the UI is disabled when using external IdP auth.

## **Configuration**

Create an .env file in the root directory (see .env.example). You **must** set the JWT_SECRET.

```
JWT_SECRET=your-super-secret-key  
WIKI_NAME="My Wiki"
DB_PATH=wiki.db
LOG_DB_PATH=logs.db
TRUST_PROXY_HEADERS=true # if behind a reverse proxy
INSECURE_COOKIES=true # if running on a local network with Docker without HTTPS
PORT=8080 # useful for Docker deployments
```

### External IdP Support

To enable external IdP support, set the following environment variables:

```
JWKS_URL=https://example.com/.well-known/jwks.json
JWT_ISSUER=https://example.com/
JWT_EMAIL_CLAIM=email # Optional, defaults to "email". Looks in ID token if present.
```

### Plugin Support

```
PLUGIN_PATH=plugins
PLUGIN_STORAGE_PATH=plugins.db
```

## **CLI Usage**

Wikilite includes a CLI for managing the system without needing the API.

### **User Management**

```
# Add a new user  
./wikilite add-user --email user@example.com --name "John Doe" --password "secret" --role write

# Update an existing user  
./wikilite update-user --email user@example.com --role admin --enable

# Remove a user  
./wikilite remove-user --email user@example.com
```

### **Maintenance**

```
# Prune system logs older than 30 days  
./wikilite prune-logs --days 30
```

## **Starting the Server**

To start the application:

```
./wikilite serve
```

* **First Run:** The system will automatically seed a default admin user:
    * **Email:** admin@example.com
    * **Password:** admin
* Home page: http://localhost:8080/.

## **API Documentation**

Wikilite provides interactive API documentation generated via Huma.

1. Start the server (`./wikilite serve`).
2. Navigate to http://localhost:8080/docs in your browser.

## **Plugins**

Wikilite supports a basic plugin system. Example plugins can be found in the `example_plugins` directory.

### Hooks
1. `onArticleRender`: Modify the HTML output of an article after it has been rendered from Markdown.
2. `onAction`: Run custom code, triggered when posting to `/api/plugin/{pluginID}/{action}`

```typescript
/**
 * Hook: onArticleRender
 * Called when the article content is ready to be rendered.
 * @param html The current HTML string of the article.
 * @param ctx The request context containing user info, etc.
 * @returns The modified HTML string.
 */
declare function onArticleRender(html: string, ctx: Context): string;

/**
 * Hook: onAction
 * Called when a client POSTs to /api/plugin/{pluginID}/{action}.
 * @param action The action name from the URL.
 * @param payload The JSON parsed body of the request.
 * @param ctx The request context containing user info, etc.
 * @returns A JSON-serializable object response.
 */
declare function onAction(action: string, payload: any, ctx: Context): any;
```

### Plugin Development
1. Create a directory for your plugins and set with the `PLUGIN_PATH` environment variable.
2. Create a .js file for each plugin. 
    * Plugins must start with "##-" and are run in numerical order.
3. Include a function named `onArticleRender` and/or `onAction` in your plugin.


### **Javascript Environment**

The plugin runtime is based on [QuickJS](https://modernc.org/quickjs). 

#### Built-in Functionality
* Private Key/Value Storage through `Host.storage`
* Logging through `console`
* HTML Sanitization through `Host.sanitize` or `DOMPurify.sanitize`

#### Provided JS Libraries
Handful of useful libraries are included by default.

* Simulated DOM - **[linkedom](https://github.com/WebReflection/linkedom)** (`DOMParser`)
* **[Lodash](https://github.com/lodash/lodash)** (`_`)
* **[Mustache](https://github.com/janl/mustache.js)** (`Mustache`)
* **[Marked](https://github.com/markedjs/marked)** (`marked`)
* **[Day.js](https://github.com/iamkun/dayjs/)** (`dayjs`)

#### **Custom Library Bundle**

If you wish to use different libraries or specific versions, you can override the default environment by providing your own bundled JavaScript file.

1. Set the `JSPKGS_PATH` environment variable to the path of your bundle.  
   ```
   JSPKGS_PATH=/path/to/custom/jspkgs.js
   ```

2. Your bundle should assign libraries to `globalThis` to make them available to plugins.

**Recommendation:** Recommend using [**esbuild**](https://esbuild.github.io/) to create your bundle for optimal performance and compatibility.

```
# Example esbuild command  
esbuild entry.js --bundle --minify --outfile=jspkgs.js --target=es2020
```  