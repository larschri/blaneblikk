# Blåneblikk

Render terrain scenery using elevation data from [Kartverket](https://www.kartverket.no/data/).

> * _blåne (norwegian) - something that is so far away that it appears blue against mountains even further away, or towards the horizon._
> * _blikk (norwegian) - view_

This software renders PNG images of scenery as
seen from a given viewpoint in the terrain.
A web server keeps elevation data in (virtual) memory
for efficient access, and renders images when requested.
There is also a crude web interface that allows selecting
viewpoint and direction of sight by clicking a map.

## Setup

* Install [GDAL](https://gdal.org/) ≥ v3.0 and make headers available in sub-folder _gdal_. On linux: `ln -s /usr/include/gdal gdal`.
* Sign up at [Kartverket](https://www.kartverket.no/data/) and download USGS DEM files from data set "DTM 10 Terrengmodell (UTM32)" into sub-folder _dem-files_.
* Install [Go](https://golang.org/doc/install)

## Getting started

At https://kartkatalog.geonorge.no/ locate dataset "DTM 10 Terrengmodell (UTM 32)".

To get started, download these two files:
* 6804\_2\_10m\_z32.dem
* 6804\_3\_10m\_z32.dem

Then start the web server
`go run . --address=localhost:4242 --demfiles=dem-files --mmapfiles=/tmp`

Then visit this URL to get a view from Galdhøpiggen towards Hurrungane
`http://localhost:4242/bb?lat0=61.63637302336104&lng0=8.312476873397829&lat1=61.461421091200464&lng1=7.8714895248413095`

![View from Galdhøpiggen towards Hurrungane](https://github.com/larschri/blaneblikk/blob/wip-something/server/static/example.png?raw=true)

©Kartverket

## Cartesian geometry

The geometry is based on cartesian coordinates in metric units (meters) as defined by the
[Universal Transverse Mercator coordinate system](https://en.wikipedia.org/wiki/Universal_Transverse_Mercator_coordinate_system).
Terrain far away are lowered to achieve a earth curvature effect.
This is a simplistic geometrical model that will cause some distortion depending on location.
