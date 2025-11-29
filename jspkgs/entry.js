/**
 * entry.js
 */

const { parseHTML } = require("linkedom");

const dom = parseHTML(`
<!doctype html>
<html lang="en">
  <head><title>QuickJS Context</title></head>
  <body></body>
</html>
`);

globalThis.window = dom.window;
globalThis.document = dom.document;
globalThis.HTMLElement = dom.HTMLElement;
globalThis.Node = dom.Node;
globalThis.Text = dom.Text;
globalThis.CustomEvent = dom.CustomEvent;
globalThis.Event = dom.Event;
globalThis.navigator = dom.navigator;
globalThis.DOMParser = dom.DOMParser;

globalThis.location = {
  href: "http://localhost/",
  protocol: "http:",
  host: "localhost",
  hostname: "localhost",
  origin: "http://localhost",
  pathname: "/",
  search: "",
  hash: "",
};

globalThis._ = require("lodash");
globalThis.Mustache = require("mustache");

const { marked } = require("marked");
globalThis.marked = marked;

globalThis.dayjs = require("dayjs");

// Polyfill DOMPurify:
globalThis.DOMPurify = {
    sanitize: (html) => {
        if (typeof Host !== 'undefined' && Host.sanitize) {
            return Host.sanitize(html);
        }
        return html;
    }
};
