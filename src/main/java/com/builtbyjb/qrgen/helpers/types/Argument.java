package com.builtbyjb.qrgen.helpers.types;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.EqualsAndHashCode;
import lombok.NoArgsConstructor;

@AllArgsConstructor
@NoArgsConstructor
@EqualsAndHashCode
@Builder
public class Argument {
    Integer quantity;
    String info;
    Integer width;
    Integer height;
    String url;
    Format format;
    Storage storage;

    @Override
    public String toString() {
        String str = String.format(
                "Argument{quantity=%d, info='%s', width=%d, height=%d, url='%s', format='%s', storage=%s}",
                quantity, info, width, height, url, format, storage);
        return str;
    }
}
