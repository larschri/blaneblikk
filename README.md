# Blåner
Blåne - noe som ligger så langt borte at det fortoner seg blått mot fjell enda lenger borte, eller mot synsranden.



# Setup

## Gdal
Install gdal on your system. Headers must be available in sub-folder _gdal_. On linux: `ln -s /usr/include/gdal gdal`.

## DEM data from kartverket

Sign up at kartverket.no and download DEM files from dataset "DTM 10 Terrengmodell (UTM32)" into sub-folder _dem-files_.

## Getting started

At https://kartkatalog.geonorge.no/ locate dataset "DTM 10 Terrengmodell (UTM 32)".

To get started, download these two files:
* 6804_2_10m_z32.dem
* 6804_3_10m_z32.dem

Then start the web server
`go run . --address=localhost:4242 --demfiles=demfiles --mmapfiles=/tmp`

Then visit this URL to get a view from Galdhøpiggen towards Hurrungane
`http://localhost:4242/blaner?lat0=61.63637302336104&lng0=8.312476873397829&lat1=61.461421091200464&lng1=7.8714895248413095`

![View from Galdhøpiggen towards Hurrungane](https://github.com/larschri/blaner/blob/wip-transform/htdocs/example.png?raw=true)

©Kartverket