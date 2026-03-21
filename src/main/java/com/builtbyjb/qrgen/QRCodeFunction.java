package com.builtbyjb.qrgen;

import com.builtbyjb.qrgen.config.PostgresConfig;
import com.builtbyjb.qrgen.helpers.Context;
import com.builtbyjb.qrgen.helpers.Argument;
import com.builtbyjb.qrgen.helpers.ParseTime;
import com.builtbyjb.qrgen.helpers.Storage;
import com.builtbyjb.qrgen.service.QRCodeService;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;

public class QRCodeFunction {

    private final QRCodeService qrCodeService;
    private final PostgresConfig dbConfig;
    private final String version = "0.1.0";

    public QRCodeFunction() {
        this.qrCodeService = new QRCodeService();
        this.dbConfig = new PostgresConfig();

        dbConfig.init();
        if (Context.DEBUG.greaterThanOrEqual(1))
            System.out.println("Database tables initialized");
    }

    public static void main(String[] args) {
        System.out.println(List.of(args));

        Optional<Argument> argument = parseArguments(args);
        if (argument.isEmpty()) {
            return;
        }

        System.out.println(argument.get());

    }

    private Optional<Argument> parseArguments(String[] args) {

        int quantity = 0; // Required
        String info = "";
        int width = 500;
        int height = 500;
        String url = ""; // Required
        String format = ""; // Required; Note: format can also be an enum
        Storage storage = Storage.fromString("local");

        for (String arg : args) {
            String[] str = arg.split("=");
            switch (str[0]) {
                case "--version", "-v":
                    System.out.println("QR gen version " + version);
                    return null;
                case "--help", "-h":
                    System.out.println("Help/Usage");
                    return null;
                case "--quantity":
                    quantity = Integer.parseInt(str[1]);
                    break;
                case "--size":
                    String[] szs = str[1].split("x");
                    width = Integer.parseInt(szs[0]);
                    height = Integer.parseInt(szs[1]);
                    break;
                case "--info":
                    info = str[1];
                    break;
                case "--url":
                    url = str[1];
                    break;
                case "--format":
                    format = str[1];
                    break;
                case "--storage":
                    storage = Storage.fromString(str[1]);
                    break;
            }
        }

        if (quantity < 1) {
            System.out.println("Error: Quantity must be greater than 0");
            // Print usage instructions
            return null;
        }

        if (url.isEmpty()) {
            System.out.println("Error: URL is required");
            // Print usage instructions
            return null;
        }

        if (format.isEmpty()) {
            System.out.println("Error: Format is required");
            // Print usage instructions
            return null;
        }

        Agrument argument = Argument.builder()
                .quantity(quantity)
                .info(info)
                .width(width)
                .height(height)
                .url(url)
                .format(format)
                .storage(storage)
                .build();

        return Optional.ofNullable(argument);
    }
}
