// Signs Swagger UI "Try it out" requests with the app credential (see docs/AUTH.md).
//
// SHA-256 / HMAC are implemented in plain JS because crypto.subtle only exists in
// secure contexts (HTTPS or localhost), and dev servers are often opened by LAN IP.
(function (root) {
  'use strict';

  var K = new Uint32Array([
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ]);

  function sha256(bytes) {
    var bitLength = bytes.length * 8;
    // Message + 0x80 + 64-bit length, rounded up to a multiple of 64 bytes.
    var paddedLength = Math.ceil((bytes.length + 9) / 64) * 64;
    var data = new Uint8Array(paddedLength);
    data.set(bytes);
    data[bytes.length] = 0x80;
    var view = new DataView(data.buffer);
    view.setUint32(paddedLength - 8, Math.floor(bitLength / 0x100000000));
    view.setUint32(paddedLength - 4, bitLength >>> 0);

    var h = new Uint32Array([
      0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
    ]);
    var w = new Uint32Array(64);

    for (var offset = 0; offset < paddedLength; offset += 64) {
      for (var i = 0; i < 16; i++) w[i] = view.getUint32(offset + i * 4);
      for (i = 16; i < 64; i++) {
        var x = w[i - 15], y = w[i - 2];
        var s0 = ((x >>> 7) | (x << 25)) ^ ((x >>> 18) | (x << 14)) ^ (x >>> 3);
        var s1 = ((y >>> 17) | (y << 15)) ^ ((y >>> 19) | (y << 13)) ^ (y >>> 10);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1) | 0;
      }

      var a = h[0], b = h[1], c = h[2], d = h[3], e = h[4], f = h[5], g = h[6], k = h[7];
      for (i = 0; i < 64; i++) {
        var S1 = ((e >>> 6) | (e << 26)) ^ ((e >>> 11) | (e << 21)) ^ ((e >>> 25) | (e << 7));
        var t1 = (k + S1 + ((e & f) ^ (~e & g)) + K[i] + w[i]) | 0;
        var S0 = ((a >>> 2) | (a << 30)) ^ ((a >>> 13) | (a << 19)) ^ ((a >>> 22) | (a << 10));
        var t2 = (S0 + ((a & b) ^ (a & c) ^ (b & c))) | 0;
        k = g; g = f; f = e; e = (d + t1) | 0; d = c; c = b; b = a; a = (t1 + t2) | 0;
      }

      h[0] += a; h[1] += b; h[2] += c; h[3] += d; h[4] += e; h[5] += f; h[6] += g; h[7] += k;
    }

    var out = new Uint8Array(32);
    var outView = new DataView(out.buffer);
    for (i = 0; i < 8; i++) outView.setUint32(i * 4, h[i]);
    return out;
  }

  function hmacSha256(keyBytes, messageBytes) {
    if (keyBytes.length > 64) keyBytes = sha256(keyBytes);
    var inner = new Uint8Array(64 + messageBytes.length);
    var outer = new Uint8Array(64 + 32);
    for (var i = 0; i < 64; i++) {
      var byte = keyBytes[i] || 0;
      inner[i] = byte ^ 0x36;
      outer[i] = byte ^ 0x5c;
    }
    inner.set(messageBytes, 64);
    outer.set(sha256(inner), 64);
    return sha256(outer);
  }

  function toHex(bytes) {
    var hex = '';
    for (var i = 0; i < bytes.length; i++) hex += (bytes[i] < 16 ? '0' : '') + bytes[i].toString(16);
    return hex;
  }

  function randomHex(size) {
    var bytes = new Uint8Array(size);
    root.crypto.getRandomValues(bytes);
    return toHex(bytes);
  }

  function findHeader(headers, name) {
    var lower = name.toLowerCase();
    for (var key in headers) {
      if (key.toLowerCase() === lower) return key;
    }
    return null;
  }

  function concatBytes(chunks) {
    var length = 0;
    for (var i = 0; i < chunks.length; i++) length += chunks[i].length;
    var out = new Uint8Array(length);
    var offset = 0;
    for (i = 0; i < chunks.length; i++) {
      out.set(chunks[i], offset);
      offset += chunks[i].length;
    }
    return out;
  }

  // Same escaping browsers apply to multipart field names and file names.
  function escapeQuoted(value) {
    return String(value).replace(/\r/g, '%0D').replace(/\n/g, '%0A').replace(/"/g, '%22');
  }

  // encodeMultipart serializes FormData with an all lower-case boundary.
  // Browsers lower-case a Blob's type, so a boundary such as Chrome's
  // "----WebKitFormBoundaryAbC" would reach the server as "...abc" while the
  // body still uses "...AbC", and the server could not find any part.
  async function encodeMultipart(formData) {
    var encoder = new TextEncoder();
    var boundary = '----securepatrol' + randomHex(16);
    var entries = [];
    formData.forEach(function (value, name) { entries.push([name, value]); });

    var chunks = [];
    for (var i = 0; i < entries.length; i++) {
      var name = entries[i][0];
      var value = entries[i][1];
      chunks.push(encoder.encode('--' + boundary + '\r\n'));

      if (typeof value === 'string') {
        chunks.push(encoder.encode('Content-Disposition: form-data; name="' + escapeQuoted(name) + '"\r\n\r\n' + value + '\r\n'));
        continue;
      }

      chunks.push(encoder.encode(
        'Content-Disposition: form-data; name="' + escapeQuoted(name) + '"; filename="' + escapeQuoted(value.name || 'blob') + '"\r\n' +
        'Content-Type: ' + (value.type || 'application/octet-stream') + '\r\n\r\n'
      ));
      chunks.push(new Uint8Array(await value.arrayBuffer()));
      chunks.push(encoder.encode('\r\n'));
    }
    chunks.push(encoder.encode('--' + boundary + '--\r\n'));

    return { bytes: concatBytes(chunks), contentType: 'multipart/form-data; boundary=' + boundary };
  }

  // Returns the exact bytes that will be sent. Form bodies are serialized here
  // (instead of by fetch) so the signed bytes and the boundary match what the
  // server receives. They are sent as a Blob because Swagger strips any explicit
  // multipart Content-Type header after this interceptor runs; fetch then takes
  // the Content-Type, boundary included, from the Blob type.
  async function takeBody(req) {
    var body = req.body;
    var encoder = new TextEncoder();

    if (body === undefined || body === null || body === '') return new Uint8Array(0);
    if (typeof body === 'string') return encoder.encode(body);

    if (body instanceof FormData) {
      var multipart = await encodeMultipart(body);
      var multipartHeader = findHeader(req.headers, 'Content-Type');
      if (multipartHeader) delete req.headers[multipartHeader];
      req.body = new Blob([multipart.bytes], { type: multipart.contentType });
      return multipart.bytes;
    }

    if (body instanceof URLSearchParams) {
      var response = new Response(body);
      var contentType = response.headers.get('content-type');
      var bytes = new Uint8Array(await response.arrayBuffer());
      var ctHeader = findHeader(req.headers, 'Content-Type');
      if (ctHeader) delete req.headers[ctHeader];
      req.body = new Blob([bytes], { type: contentType });
      return bytes;
    }

    if (body instanceof Blob) return new Uint8Array(await body.arrayBuffer());
    if (body instanceof ArrayBuffer) return new Uint8Array(body);
    if (ArrayBuffer.isView(body)) return new Uint8Array(body.buffer, body.byteOffset, body.byteLength);

    req.body = JSON.stringify(body);
    return encoder.encode(req.body);
  }

  // signRequest adds X-Timestamp, X-Nonce and X-Signature to an /api/ request
  // that carries X-App-Id and the (docs only) X-App-Key header. X-App-Key is
  // removed so the key never leaves the browser.
  async function signRequest(req) {
    req.headers = req.headers || {};
    var url = new URL(req.url, root.location ? root.location.href : undefined);
    if (url.pathname.indexOf('/api/') !== 0) return req;

    var keyHeader = findHeader(req.headers, 'X-App-Key');
    if (!keyHeader) return req;
    var appKey = req.headers[keyHeader];
    delete req.headers[keyHeader];
    if (!appKey || !findHeader(req.headers, 'X-App-Id')) return req;

    var encoder = new TextEncoder();
    var body = await takeBody(req);
    var timestamp = Math.floor(Date.now() / 1000).toString();
    var nonce = randomHex(16);
    var payload = [
      (req.method || 'GET').toUpperCase(),
      url.pathname + url.search,
      timestamp,
      nonce,
      toHex(sha256(body)),
    ].join('\n');

    req.headers['X-Timestamp'] = timestamp;
    req.headers['X-Nonce'] = nonce;
    req.headers['X-Signature'] = toHex(hmacSha256(encoder.encode(appKey), encoder.encode(payload)));
    return req;
  }

  root.SecurePatrolSigner = { signRequest: signRequest, sha256: sha256, hmacSha256: hmacSha256, toHex: toHex };
})(typeof window !== 'undefined' ? window : globalThis);
