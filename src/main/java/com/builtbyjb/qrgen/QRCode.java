package com.builtbyjb.qrgen;

import com.builtbyjb.qrgen.config.PostgresConfig;
import com.builtbyjb.qrgen.helpers.Context;
import com.builtbyjb.qrgen.helpers.Parser;
import com.builtbyjb.qrgen.helpers.types.Argument;
import com.builtbyjb.qrgen.service.QRCodeService;
import java.util.Optional;

public class QRCode {

    private final PostgresConfig dbConfig;
    private static final String version = "0.1.0";

    public QRCode() {
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

        try {
            boolean result = new QRCodeService().generateQRCodes(argument.get());
            if (!result) {
                System.out.println("Failed to generate QR codes");
                return;
            }
        } catch (Exception e) {
            System.out.println("Error generating QR codes: " + e.getMessage());
            return;
        }

        System.out.println("QR codes generated successfully");
    }
}
