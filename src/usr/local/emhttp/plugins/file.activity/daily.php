#!/usr/bin/php -q
<?php

namespace FileActivity;

$fileRoot = dirname(__FILE__);

require_once "{$fileRoot}/include/common.php";

if ( ! defined(__NAMESPACE__ . '\PLUGIN_NAME')) {
    throw new \RuntimeException("PLUGIN_NAME not defined");
}

Utils::run_task(__NAMESPACE__ . '\Utils::sendUsageData');
