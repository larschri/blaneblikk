var map = L.map('map').setView([60.14, 10.25], 11);
L.tileLayer('https://opencache.statkart.no/gatekeeper/gk/gk.open_gmaps?layers=topo4&zoom={z}&x={x}&y={y}', {
	attribution: '<a href="http://www.kartverket.no/">Kartverket</a>'
}).addTo(map);

var marker = null;
var marker2 = L.marker();

function onMapClick(e) {
	if (marker == null) {
	    marker = L.marker();
	    marker
		.setLatLng(e.latlng)
		.addTo(map);
	} else {
	    marker2
		.setLatLng(e.latlng)
		.addTo(map);
		pos0 = marker.getLatLng();
		pos1 = marker2.getLatLng();
		document.querySelector("#bbImg").src = `bb?lat0=${pos0.lat}&lng0=${pos0.lng}&lat1=${pos1.lat}&lng1=${pos1.lng}`;
	}
}

function setPos(latlng) {
	map.panTo(latlng)
	if (marker == null) {
		marker = L.marker();
	}
	marker
		.setLatLng(latlng)
		.addTo(map);
	marker2.remove();
}

map.on('click', onMapClick);

document.querySelector('#reset').addEventListener('click', event => {
	if (marker != null) {
		marker.remove();
		marker = null;
	}
	marker2.remove();
});

document.querySelector('#bbImg').addEventListener('click', event => {
    let url = new URL(event.srcElement.src);
    url.pathname = "bb/pixelLatLng"
    url.search += "&offsetX=" + event.offsetX + "&offsetY=" + event.offsetY;
    let request = new XMLHttpRequest();
    request.open("GET", url.toString());
    request.responseType = "json";
    request.send();
    request.onload = function() {
	marker2
		.setLatLng(request.response)
		.addTo(map);
	map.panInside(request.response)
    };
});
