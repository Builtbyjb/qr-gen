package com.pesira.traceability.config;

import com.pesira.traceability.helpers.Context;
import io.github.cdimascio.dotenv.Dotenv;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.Optional;

public class PostgresConfig {

    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();
    private final String DB_NAME;
    private final String DB_USER;
    private final String DB_PASSWORD;
    private final String DB_HOST;
    private final String CONNECTION_NAME;
    private final String QR_CODE_TABLE_NAME;

    public PostgresConfig() {
        DB_NAME = Optional.ofNullable(System.getenv("DB_NAME")).orElse(dotenv.get("DB_NAME"));
        DB_USER = Optional.ofNullable(System.getenv("DB_USER")).orElse(dotenv.get("DB_USER"));
        DB_PASSWORD = Optional.ofNullable(System.getenv("DB_PASSWORD")).orElse(dotenv.get("DB_PASSWORD"));
        DB_HOST = Optional.ofNullable(System.getenv("DB_HOST")).orElse(dotenv.get("DB_HOST"));
        CONNECTION_NAME = Optional.ofNullable(System.getenv("CONNECTION_NAME")).orElse(dotenv.get("CONNECTION_NAME"));
        QR_CODE_TABLE_NAME = Optional.ofNullable(System.getenv("QR_CODE_TABLE_NAME")).orElse(
            dotenv.get("QR_CODE_TABLE_NAME")
        );

        if (DB_NAME == null || DB_NAME.isEmpty()) {
            throw new IllegalArgumentException("ENV variable DB_NAME not found");
        }

        if (DB_USER == null || DB_USER.isEmpty()) {
            throw new IllegalArgumentException("ENV variable DB_USER not found");
        }

        if (DB_PASSWORD == null || DB_PASSWORD.isEmpty()) {
            throw new IllegalArgumentException("ENV variable DB_PASSWORD not found");
        }

        if (DB_HOST == null || DB_HOST.isEmpty()) {
            throw new IllegalArgumentException("ENV variable DB_HOST not found");
        }

        if (CONNECTION_NAME == null || CONNECTION_NAME.isEmpty()) {
            throw new IllegalArgumentException("ENV variable CONNECTION_NAME not found");
        }

        if (QR_CODE_TABLE_NAME == null || QR_CODE_TABLE_NAME.isEmpty()) {
            throw new IllegalArgumentException("ENV variable QR_CODE_TABLE_NAME not found");
        }
    }

    public Connection getConnection() {
        try {
            Class.forName("org.postgresql.Driver");

            String url = String.format(
                "jdbc:postgresql:///%s?socketFactory=com.google.cloud.sql.postgres.SocketFactory&cloudSqlInstance=%s",
                DB_NAME,
                CONNECTION_NAME
            );

            return DriverManager.getConnection(url, DB_USER, DB_PASSWORD);
        } catch (SQLException e) {
            throw new RuntimeException("Error initializing database: " + e.getMessage());
        } catch (ClassNotFoundException e) {
            throw new RuntimeException("PostgreSQL driver not found: " + e.getMessage());
        }
    }

    public void init() {
        String createQRCodesTableSql = String.format(
            """
            CREATE TABLE IF NOT EXISTS %s (
                id BIGSERIAL PRIMARY KEY,
                code_value VARCHAR(15) UNIQUE NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                processed BOOLEAN NOT NULL DEFAULT false,
                status VARCHAR(50) DEFAULT 'AVAILABLE',
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                location VARCHAR(50),
                project_id VARCHAR(50) NOT NULL,
                geolocation_id BIGINT,
                assigned VARCHAR(200)
            )
            """,
            QR_CODE_TABLE_NAME
        );

        Statement stmt = null;
        try {
            stmt = getConnection().createStatement();
            stmt.execute(createQRCodesTableSql);
        } catch (SQLException e) {
            throw new RuntimeException("Error creating QR Codes table: " + e.getMessage());
        } finally {
            try {
                if (stmt != null) {
                    stmt.close();
                    if (Context.DEBUG.greaterThanOrEqual(1)) System.out.println("Database query statement closed");
                }
                getConnection().close();
                if (Context.DEBUG.greaterThanOrEqual(1)) System.out.println("Database connection closed");
            } catch (SQLException e) {
                System.err.println("Error closing database resources: " + e.getMessage());
            }
        }
    }
}
