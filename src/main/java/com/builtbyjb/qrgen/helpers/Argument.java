package com.builtbyjb.qrgen.helpers;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.NoArgsConstructor;

@AllArgsConstructor
@NoArgsConstructor
@Builder
public class Argument {
    Integer quantity;
    String info;
    Integer width;
    Integer height;
    String url;
    String format;
    Storage storage;

    @override
    public String toString() {
        String str = String.format(
                "Argument{quantity=%d, info='%s', width=%d, height=%d, url='%s', format='%s', storage=%s}",
                quantity, info, width, height, url, format, storage
        );
        return str;
    }
}
