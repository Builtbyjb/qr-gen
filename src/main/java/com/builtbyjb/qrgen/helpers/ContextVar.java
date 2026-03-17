package com.builtbyjb.qrgen.helpers;

import java.util.HashMap;
import java.util.Map;

public class ContextVar {

    private static final Map<String, ContextVar> cache = new HashMap<>();

    private final String key;
    private final int value;

    public ContextVar(String key, int defaultValue) {
        if (cache.containsKey(key)) {
            throw new RuntimeException("Attempt to recreate ContextVar " + key);
        }

        cache.put(key, this);

        String envVar = System.getenv(key);
        if (envVar == null || envVar.isEmpty()) {
            this.value = defaultValue;
        } else {
            this.value = Integer.parseInt(envVar);
        }
        this.key = key;
    }

    public boolean getAsBoolean() {
        return this.value != 0;
    }

    public boolean greaterThan(int x) {
        return (int) this.value > x;
    }

    public boolean greaterThanOrEqual(int x) {
        return this.value >= x;
    }

    public boolean lessThan(int x) {
        return this.value < x;
    }

    public boolean lessThanOrEqual(int x) {
        return this.value <= x;
    }

    public boolean equalsValue(int x) {
        return this.value == x;
    }

    public int getValue() {
        return value;
    }

    public String getKey() {
        return key;
    }

    @Override
    public String toString() {
        return "ContextVar(key='" + key + "', value=" + value + ")";
    }
}
