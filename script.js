function createJwt2() {
  const header = Utilities.base64Encode(
    JSON.stringify({
      alg: "HS256",
      typ: "JWT",
    })
  ).replace(/=+$/, "");

  const now = Date.now();
  let expires = new Date(now);
  expires.setMinutes(expires.getMinutes() + 5);
  const payload = Utilities.base64Encode(
    JSON.stringify({
      exp: Math.round(expires.getTime() / 1000),
      iat: Math.round(now / 1000),
    //   iss: "ecommerce.site",
    //   sub: "ecommerce.user",
    })
  ).replace(/=+$/, "");
  const secretKey = "bVfpSjr...MY";
  const hexFromStr = Utilities.newBlob(secretKey)
    .getBytes()
    .map((byte) => ("0" + (byte & 0xff).toString(16)).slice(-2))
    .join("");

  const bytesFromHex = hexFromStr
    .match(/.{2}/g)
    .map((e) =>
      parseInt(e[0], 16).toString(2).length == 4
        ? parseInt(e, 16) - 256
        : parseInt(e, 16)
    );

  const toSign = Utilities.newBlob(`${header}.${payload}`).getBytes();
  const signatureBytes = Utilities.computeHmacSha256Signature(
    toSign,
    bytesFromHex
  );
  const signature = Utilities.base64EncodeWebSafe(signatureBytes).replace(
    /=+$/,
    ""
  );
  const jwt = `${header}.${payload}.${signature}`;
  console.log({
    jwt,
  });
  return jwt;
}
createJwt2();