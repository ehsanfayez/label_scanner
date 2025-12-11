<?php
// Example usage
try {
    $secret = hex2bin("b2e32f9dd1a8c4e9ef6b339c8c373eab85a9cda934f3dfc2b88d7c5c4bb1e8f0");
    $data = json_encode([
        "inventory_id" => "12345ABC",
        "serial_number" => "S98765"
    ]);

    $key = substr($secret, 0, 32);
    $iv = random_bytes(12);

    $cipher = openssl_encrypt(
        $data,
        'aes-256-gcm',
        $key,
        OPENSSL_RAW_DATA,
        $iv,
        $tag
    );

    function b64u($d){ return rtrim(strtr(base64_encode($d), '+/', '-_'), '='); }

    $token = b64u($iv) . "." . b64u($cipher) . "." . b64u($tag);

    echo $token;
} catch (Exception $e) {
    die("Error initializing CryptoService: " . $e->getMessage());
}


