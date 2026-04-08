function handleOnline(msg) {
  updateOverlay(true, msg);
  if (globalThis.onlineFunc) {
    globalThis.onlineFunc();
  }
}

function handleOffline(msg) {
  updateOverlay(false, msg);
  if (globalThis.offlineFunc) {
    globalThis.offlineFunc();
  }
}

async function loadWasm() {
  const go = new Go(); // Defined in wasm_exec.js

  // Show the loading overlay and apply the background effects
  updateOverlay(false, "Loading...");

  // Initialize the environment variables
  globalThis.go_env = {};

  // Define the pause and resume functions
  globalThis.offlineFunc = function () {
    console.log("Not implemented: offlineFunc");
  };

  globalThis.onlineFunc = function () {
    console.log("Not implemented: onlineFunc");
  };

  try {
    // Load and instantiate the WebAssembly module
    const wasmModule = await WebAssembly.instantiateStreaming(
      fetch("main.wasm"),
      go.importObject,
    );

    // Change overlay text to "Running..."
    updateOverlay(true, "Game loaded. Running...");

    // Introduce a small delay to ensure the DOM is updated before running the Wasm module
    await new Promise((resolve) => setTimeout(resolve, 100)); // 100ms delay

    // Run the WebAssembly module
    go.run(wasmModule.instance);
  } catch (error) {
    console.error("Failed to load WASM module:", error);
    handleOffline("Failed to load game. Check your connection and try again.");
    return;
  }

  const worker = new Worker("health-worker.js");

  worker.onmessage = function (e) {
    switch (e.data.type) {
      case "offline":
        handleOffline("Game server is down. Waiting for connection...");
        break;
      case "online":
        handleOnline("Game server is up. Resuming...");
        break;
    }
    worker.postMessage({ type: "ack" });
  };

  worker.postMessage({ type: "start" });
}

function updateOverlay(activate, msg) {
  const overlay = document.getElementById("loadingOverlay");
  const overlayText = document.getElementById("loadingMessage");
  overlayText.innerText = msg;

  if (activate) {
    overlay.classList.add("hidden");
    overlay.classList.remove("active");
  } else {
    overlay.classList.add("active");
    overlay.classList.remove("hidden");
  }
}

window.addEventListener("load", loadWasm);

window.addEventListener("online", () => {
  handleOnline("Connection restored. Resuming...");
});

window.addEventListener("offline", () => {
  handleOffline("Connection lost. Waiting for connection...");
});

if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register("service-worker.js").then(
    (registration) => {
      console.log(
        "ServiceWorker registration successful with scope: ",
        registration.scope,
      );
    },
    (error) => {
      console.log("ServiceWorker registration failed: ", error);
    },
  );
}
