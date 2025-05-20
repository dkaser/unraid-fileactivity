<?php

namespace FileActivity;

class Translator
{
    /** @var array<string, string> $lang */
    private array $lang;

    /**
     * @return array<string, string>
     * @param array<mixed,mixed> $array
     */
    private static function flattenArray(array $array, string $keys = ""): array
    {
        $result = array();
        foreach ($array as $key => $value) {
            if (is_array($value)) {
                $result = array_merge($result, self::flattenArray($value, $keys . $key . "."));
            } else {
                if (is_string($value) && ! empty($value)) {
                    $result[$keys . $key] = $value;
                }
            }
        }
        return $result;
    }

    public function __construct()
    {
        global $login_locale;

        if ( ! defined(__NAMESPACE__ . "\PLUGIN_ROOT")) {
            throw new \RuntimeException("PLUGIN_ROOT not defined");
        }

        $dynamix = parse_ini_file('/boot/config/plugins/dynamix/dynamix.cfg', true) ?: array();

        $locale        = $_SESSION['locale'] ?? ($login_locale ?? ($dynamix['display']['locale'] ?? "none"));
        $plugin_locale = (array) json_decode(file_get_contents(PLUGIN_ROOT . "/locales/en_US.json") ?: "{}", true);
        $plugin_lang   = self::flattenArray($plugin_locale);

        if (file_exists(PLUGIN_ROOT . "/locales/{$locale}.json")) {
            $current_locale = (array) json_decode(file_get_contents(PLUGIN_ROOT . "/locales/{$locale}.json") ?: "{}", true);
            $current_lang   = self::flattenArray($current_locale);
            $plugin_lang    = array_replace($plugin_lang, $current_lang);
        }

        $this->lang = $plugin_lang;
    }

    public function tr(string $message): string
    {
        return $this->lang[strtolower($message)];
    }
}
