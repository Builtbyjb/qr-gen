package com.builtbyjb.qrgen.helpers.types;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.EqualsAndHashCode;
import lombok.Getter;
import lombok.NoArgsConstructor;

@AllArgsConstructor
@NoArgsConstructor
@EqualsAndHashCode
@Builder
@Getter
public class Argument {
    Integer quantity;
    String info;
    Integer size;
    String url;
    Format format;
    Storage storage;

    @Override
    public String toString() {
        String str = String.format(
                "Argument{quantity=%d, info='%s', size=%d, url='%s', format='%s', storage=%s}",
                quantity, info, size, url, format, storage);
        return str;
    }
}
