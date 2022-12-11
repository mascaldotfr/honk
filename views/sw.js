self.addEventListener('install', function (event) {
	console.log('service worker installed')
})
self.addEventListener('fetch',() => console.log("fetch"));
