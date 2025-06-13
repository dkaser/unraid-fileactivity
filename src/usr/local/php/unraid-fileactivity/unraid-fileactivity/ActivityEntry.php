<?php

namespace EDACerton\FileActivity;

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

class ActivityEntry
{
    private string $timestamp;
    private string $action;
    private string $filePath;

    public function __construct(string $line)
    {
        $data = str_getcsv($line);

        $this->timestamp = $data[0] ?? "";
        $this->action    = $data[1] ?? "";
        $this->filePath  = $data[2] ?? "";
    }

    public function getTimestamp(): string
    {
        return $this->timestamp;
    }

    public function getAction(): string
    {
        return $this->action;
    }

    public function getFilePath(): string
    {
        return $this->filePath;
    }

    /**
     * @return array<string, string>
     */
    public function toArray(): array
    {
        return [
            'timestamp' => $this->getTimestamp(),
            'action'    => $this->getAction(),
            'filePath'  => $this->getFilePath()
        ];
    }
}
