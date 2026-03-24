package com.builtbyjb.qrgen.helpers.types;

public enum Storage {
    LOCAL,
    AWS_S3,
    GOOGLE_CLOUD_STORAGE,
    AZURE_BLOB_STORAGE,
    GOOGLE_DRIVE,
    DROPBOX,
    ONE_DRIVE;

    public static Storage fromString(String value) {
        try {
            return Storage.valueOf(value.toUpperCase());
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException("Invalid storage option: " + value);
        }

    }
}
