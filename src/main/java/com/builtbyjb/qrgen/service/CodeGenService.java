package com.builtbyjb.qrgen.service;

import io.github.cdimascio.dotenv.Dotenv;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Optional;

public class CodeGenService {

    private static final String BASE62_CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
    private static final int BASE = 62;
    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();

    public String generateQRCode(long cursor) {
        String qrCode = generateBase62(cursor, 7);
        String hash = generateHash().toUpperCase();
        String qrCodeWithHash = randomInsertHash(qrCode, hash, cursor);
        return "QR-" + qrCodeWithHash;
    }

    public String generateBase62(long cursor, int length) {
        char[] buf = new char[length];
        int lastIndex = length - 1;

        while (cursor > 0 && lastIndex >= 0) {
            // Decreases lastIndex in every iteration
            buf[lastIndex--] = BASE62_CHARS.charAt((int) (cursor % BASE));
            cursor /= BASE;
        }

        // Pad remaining values with zero
        while (lastIndex >= 0)
            buf[lastIndex--] = BASE62_CHARS.charAt(0);

        return new String(buf);
    }

    private String generateHash() {
        String salt = Optional.ofNullable(System.getenv("SECRET_KEY")).orElse(dotenv.get("SECRET_KEY"));
        if (salt == null || salt.isEmpty())
            throw new IllegalArgumentException("ENV variable SECRET_KEY not found");

        try {
            // Hash with SHA-256
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(salt.getBytes(StandardCharsets.UTF_8));

            // Convert hash bytes into Base62 string
            StringBuilder base62 = new StringBuilder();
            for (byte b : hash) {
                int unsigned = b & 0xFF;
                base62.append(BASE62_CHARS.charAt(unsigned % BASE62_CHARS.length()));
            }

            // Take only the first N characters for the QR code
            String code = base62.substring(0, 3);

            return code;
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("Failed to generate hash", e);
        }
    }

    private String randomInsertHash(String qrCode, String hash, long cursor) {
        int position = (int) cursor % 5;

        switch (position) {
            case 0:
                return qrCode.substring(2, 7) + hash + qrCode.substring(0, 2);
            case 1:
                return qrCode.substring(0, 3) + hash + qrCode.substring(3, 7);
            case 2:
                return qrCode.substring(3, 7) + hash + qrCode.substring(0, 3);
            case 4:
                return qrCode + hash;
            case 5:
                return hash + qrCode;
            default:
                return hash + qrCode;
        }
    }
}
