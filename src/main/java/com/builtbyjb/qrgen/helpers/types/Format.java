package com.builtbyjb.qrgen.helpers.types;

public enum Format {
    PDF,
    PNG,
    SVG,
    JPEG;

    public static Format fromString(String value) {
        try {
            return Format.valueOf(value.toUpperCase());
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException("Invalid format option: " + value);
        }
    }
}
