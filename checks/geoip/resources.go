package geoip

import _ "embed"

//go:embed country-to-continent.json
var CountryToContinentJSON []byte
