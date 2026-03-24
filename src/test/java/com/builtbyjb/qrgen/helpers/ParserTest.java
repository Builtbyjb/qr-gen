package com.builtbyjb.qrgen.helpers;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import com.builtbyjb.qrgen.helpers.types.Argument;
import com.builtbyjb.qrgen.helpers.types.Format;
import com.builtbyjb.qrgen.helpers.types.Storage;

public class ParserTest {

    @Test
    @DisplayName("Test parse time")
    public void testParseTime() {
        Assertions.assertEquals("500.00 μs", Parser.parseTime(0.5));
        Assertions.assertEquals("500.00 ms", Parser.parseTime(500));
        Assertions.assertEquals("1.00 s", Parser.parseTime(1000));
        Assertions.assertEquals("1 min 0 s", Parser.parseTime(60_000));
        Assertions.assertEquals("1 h 0 min", Parser.parseTime(3_600_000));
        Assertions.assertEquals("1.00 days", Parser.parseTime(86_400_000));
    }

    @Test
    @DisplayName("Test parse arguments")
    public void testParseArguments() {
        String[] args = { "--quantity=10", "--info=Test QR Code", "--size=500x500",
                "--url=https://example.com", "--format=png", "--storage=local" };
        Argument expected = Argument.builder()
                .quantity(10)
                .info("Test QR Code")
                .width(500)
                .height(500)
                .url("https://example.com")
                .format(Format.PDF)
                .storage(Storage.LOCAL)
                .build();
        Argument actual = Parser.parseArguments(args, "0.1.0").orElse(null);
        Assertions.assertEquals(expected, actual);
    }
}
