#!/usr/bin/php -q
<?php

namespace EDACerton\FileActivity;

use EDACerton\PluginUtils\Utils;

$docroot = $docroot ?? $_SERVER['DOCUMENT_ROOT'] ?: '/usr/local/emhttp';
require_once dirname(dirname(__FILE__)) . "/include/common.php";

$utils = new Utils("fileactivity");

$new_config = '/boot/config/plugins/file.activity/config.json';
$old_config = '/boot/config/plugins/file.activity/file.activity.cfg';

if (file_exists($old_config) && ! file_exists($new_config)) {
    // Migrate old config to new format
    $utils->logmsg("Migrating old file.activity config.");
    $config = new Config(false);
    $config->fromINIFile();
    $config->save();
} else {
    $utils->logmsg("No config migration needed, old config file not found or new config already exists.");
}
