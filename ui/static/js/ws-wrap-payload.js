(function () {
  function handler(evt) {
    try {
      const orig = JSON.parse(evt.detail.message);

      const headers = orig.HEADERS || {};
      delete orig.HEADERS;

      const wrapped = JSON.stringify({ payload: orig, HEADERS: headers });

      evt.preventDefault();

      evt.detail.socketWrapper.sendImmediately(wrapped);
    } catch (_) {
      // If it isn't JSON, let HTMX proceed
    }
  }

  if (window.htmx) {
    htmx.defineExtension('ws-wrap-payload', {
      onEvent: function (name, evt) {
        if (name === 'htmx:wsBeforeSend') handler(evt);
      }
    });
  } else {
    // Fallback if loaded before htmx
    document.addEventListener('htmx:wsBeforeSend', handler, true);
  }
})();