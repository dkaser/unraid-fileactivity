<?php

namespace EDACerton\FileActivity;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;
use EDACerton\PluginUtils\Utils;

/*
    Copyright (C) 2025  Derek Kaser

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

require_once dirname(__FILE__) . "/include/common.php";

$prefix = "/plugins/file.activity/data.php";

if ( ! defined(__NAMESPACE__ . '\PLUGIN_ROOT') || ! defined(__NAMESPACE__ . '\PLUGIN_NAME')) {
    throw new \RuntimeException("Common file not loaded.");
}

$app = AppFactory::create();
$app->addRoutingMiddleware();
$errorMiddleware = $app->addErrorMiddleware(true, true, true);

$app->post("{$prefix}/default", function (Request $request, Response $response, $args) {
    // Reset the config to default values
    $config = new Config(false);
    $config->save();

    $utils           = new Utils("fileactivity");
    $restart_command = '/usr/local/emhttp/plugins/file.activity/scripts/rc.file.activity update';
    $utils->run_command($restart_command);

    return $response
        ->withHeader('Location', '/Tools/FileActivity')
        ->withStatus(303);
});

$app->post("{$prefix}/config", function (Request $request, Response $response, $args) {
    // We can't POST JSON directly from the WebGUI, it has to be submitted as form data due to the CSRF token.
    $data   = (array) $request->getParsedBody();
    $config = new Config();

    $config->setEnable(isset($data['enable']) ? filter_var($data['enable'], FILTER_VALIDATE_BOOLEAN) : $config->isEnabled());
    $config->setUnassignedDevices(isset($data['unassigned_devices']) ? filter_var($data['unassigned_devices'], FILTER_VALIDATE_BOOLEAN) : $config->isUnassignedDevicesEnabled());
    $config->setCache(isset($data['cache']) ? filter_var($data['cache'], FILTER_VALIDATE_BOOLEAN) : $config->isCacheEnabled());
    $config->setSSD(isset($data['ssd']) ? filter_var($data['ssd'], FILTER_VALIDATE_BOOLEAN) : $config->isSSDEnabled());
    $config->setDisplayEvents(isset($data['display_events']) && is_numeric($data['display_events']) ? intval($data['display_events']) : $config->getDisplayEvents());
    if (isset($data['exclusions']) && is_array($data['exclusions'])) {
        $exclusions = [];
        foreach ($data['exclusions'] as $exclusion) {
            if (is_string($exclusion) && ! empty($exclusion)) {
                $exclusions[] = trim($exclusion);
            }
        }
        $config->setExclusions($exclusions);
    }
    $config->setMaxRecords(isset($data['max_records']) && is_numeric($data['max_records']) ? intval($data['max_records']) : $config->getMaxRecords());
    $config->save();

    $utils           = new Utils("fileactivity");
    $restart_command = '/usr/local/emhttp/plugins/file.activity/scripts/rc.file.activity update';
    $utils->run_command($restart_command);

    return $response
        ->withHeader('Location', '/Tools/FileActivity')
        ->withStatus(303);
});

$app->get("{$prefix}/share", function (Request $request, Response $response, $args) {
    $activity = new Activity();
    $payload  = json_encode($activity->flattenActivity($activity->getShareActivity()), JSON_PRETTY_PRINT) ?: "{}";
    $response->getBody()->write($payload);
    return $response->withHeader('Content-Type', 'application/json');
});

$app->get("{$prefix}/disk", function (Request $request, Response $response, $args) {
    $activity = new Activity();
    $payload  = json_encode($activity->flattenActivity($activity->getDiskActivity()), JSON_PRETTY_PRINT) ?: "{}";
    $response->getBody()->write($payload);
    return $response->withHeader('Content-Type', 'application/json');
});

$app->get("{$prefix}/locales", function (Request $request, Response $response, $args) {
    // Get the list of supported locales from /locales (each locale is a JSON file)
    $localesDir = PLUGIN_ROOT . '/locales';
    $files      = scandir($localesDir);

    foreach ($files as $key => $file) {
        if ( ! is_file($localesDir . '/' . $file) || pathinfo($file, PATHINFO_EXTENSION) !== 'json') {
            unset($files[$key]);
        } else {
            $files[$key] = basename($file, '.json'); // Store only the locale name without extension
        }
    }

    $response->getBody()->write(json_encode($files) ?: "[]");
    return $response
        ->withHeader('Content-Type', 'application/json')
        ->withHeader('Cache-Control', 'no-store')
        ->withStatus(200);
});

$app->run();
