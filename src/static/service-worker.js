self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open("v1")
      .then((cache) => {
        // Pre-cache some static assets
        return cache.addAll([
          "/",
          "/favicon.ico",
          "/health-worker.js",
          "/index.html",
          "/manifest.json",
          "/style.css",
          "/wasm.js",
          "/audio/enemy_destroyed.wav",
          "/audio/enemy_hit.wav",
          "/audio/spaceship_acceleration.wav",
          "/audio/spaceship_boost.wav",
          "/audio/spaceship_cannon_fire.wav",
          "/audio/spaceship_crash.wav",
          "/audio/spaceship_deceleration.wav",
          "/audio/spaceship_freeze.wav",
          "/audio/spaceship_whoosh.wav",
          "/audio/theme_heroic.wav",
          "/icons/icon-192x192.png",
          "/icons/icon-512-512.png",
          "/external/ajax/libs/font-awesome/6.0.0/css/all.min.css",
        ]);
      })
      .catch((error) => {
        console.error("Failed to pre-cache assets:", error);
      }),
  );
});

self.addEventListener("fetch", (event) => {
  const isEnvRequest = event.request.url.includes("/.env");
  const isScoresRequest = event.request.url.includes("/scores.db");
  const isHealthRequest = event.request.url.includes("/health");

  if (
    event.request.method === "GET" &&
    !isEnvRequest &&
    !isScoresRequest &&
    !isHealthRequest
  ) {
    event.respondWith(
      caches.match(event.request).then((cachedResponse) => {
        if (cachedResponse) {
          const etag = cachedResponse.headers.get("ETag");

          const headers = new Headers();
          if (etag) {
            headers.append("If-None-Match", etag);
          }

          return fetch(event.request, { headers })
            .then((networkResponse) => {
              if (networkResponse.status === 304) {
                return cachedResponse;
              }

              // Update the cache with the new response
              return caches.open("v1").then((cache) => {
                cache.put(event.request, networkResponse.clone());
                return networkResponse;
              });
            })
            .catch((error) => {
              console.error(
                "Network request failed, returning cached response:",
                error,
              );
              return cachedResponse; // Return cached response if network fails
            });
        }

        // If no cached response, fetch from the network
        return fetch(event.request)
          .then((networkResponse) => {
            return caches.open("v1").then((cache) => {
              cache.put(event.request, networkResponse.clone());
              return networkResponse;
            });
          })
          .catch((error) => {
            console.error("Fetch failed and no cache available:", error);
            return new Response("Service is unavailable", {
              status: 503,
              statusText: "Service Unavailable",
            });
          });
      }),
    );
  } else {
    event.respondWith(
      fetch(event.request)
        .then((response) => response)
        .catch((error) => {
          console.error("Fetch request failed:", error);
          return new Response(JSON.stringify({ error: error.message }), {
            status: 503,
            statusText: "Service Unavailable",
            headers: { "Content-Type": "application/json" },
          });
        }),
    );
  }
});
