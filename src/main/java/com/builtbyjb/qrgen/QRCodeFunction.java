package com.builtbyjb.qrgen;

import com.builtbyjb.qrgen.config.PostgresConfig;
import com.builtbyjb.qrgen.helpers.Context;
import com.builtbyjb.qrgen.helpers.Parser;
import com.builtbyjb.qrgen.helpers.types.Argument;
import com.builtbyjb.qrgen.service.QRCodeService;
import java.util.List;
import java.util.Optional;

public class QRCodeFunction {

    private final PostgresConfig dbConfig;
    private static final String version = "0.1.0";

    public QRCodeFunction() {
        this.dbConfig = new PostgresConfig();

        dbConfig.init();
        if (Context.DEBUG.greaterThanOrEqual(1))
            System.out.println("Database tables initialized");
    }

    public static void main(String[] args) {
        Optional<Argument> argument = Parser.parseArguments(args, version);
        if (argument.isEmpty()) {
            return;
        }

        boolean result = new QRCodeService().generateQRCodes(argument.get());
        if (!result) {
            System.out.println("Failed to generate QR codes");
            return;
        }

        System.out.println("QR codes generated successfully");
    }
}
