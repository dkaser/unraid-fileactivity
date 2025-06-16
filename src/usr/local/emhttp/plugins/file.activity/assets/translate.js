/**
 * Translator class for loading and merging locale files in the browser.
 * Usage:
 *   const translator = new Translator('/plugin/path');
 *   await translator.init();
 *   translator.tr('some.key');
 */
class Translator {
  /**
   * @param {string} basePath - Path to the plugin directory (no trailing slash)
   */
  constructor(basePath) {
    this.basePath = basePath;
    this.lang = {};
  }

  /**
   * Flattens a nested object into dot-separated keys.
   * @param {object} obj
   * @param {string} prefix
   * @returns {object}
   */
  static flattenObject(obj, prefix = "") {
    let result = {};
    for (const [key, value] of Object.entries(obj)) {
      if (value && typeof value === "object" && !Array.isArray(value)) {
        Object.assign(
          result,
          Translator.flattenObject(value, prefix + key + ".")
        );
      } else if (typeof value === "string" && value.trim() !== "") {
        result[prefix + key] = value;
      }
    }
    return result;
  }

  /**
   * Loads and merges en_US and current locale files.
   * @returns {Promise<void>}
   */
  async init() {
    // Try to get browser locale, fallback to en_US
    let locale = (navigator.language || "en-US").replace("-", "_");
    if (!locale) locale = "en_US";

    const enUSPath = `${this.basePath}/locales/en_US.json`;
    const localePath = `${this.basePath}/locales/${locale}.json`;

    // Helper to fetch and parse JSON, returns {} on error
    async function fetchJson(path) {
      try {
        const res = await fetch(path);
        if (!res.ok) return {};
        return await res.json();
      } catch {
        return {};
      }
    }

    const enUS = await fetchJson(enUSPath);
    const enUSFlat = Translator.flattenObject(enUS);

    let localeFlat = {};
    if (locale !== "en_US") {
      const localeObj = await fetchJson(localePath);
      localeFlat = Translator.flattenObject(localeObj);
    }

    // Merge: localeFlat overrides enUSFlat if key exists and is not blank
    this.lang = { ...enUSFlat, ...localeFlat };
  }

  /**
   * Retrieves a translation for the given key.
   * @param {string} message
   * @param {boolean} htmlencode
   * @returns {string}
   */
  tr(message, htmlencode = true) {
    const key = message.toLowerCase();
    let value = this.lang[key] || message;
    if (htmlencode) {
      value = value.replace(/[&<>"']/g, function (c) {
        return {
          "&": "&amp;",
          "<": "&lt;",
          ">": "&gt;",
          '"': "&quot;",
          "'": "&#39;",
        }[c];
      });
    }
    return value;
  }
}

// Make globally accessible for use in HTML <script> tags
window.Translator = Translator;
