package com.builtbyjb.qrgen.repository;

import com.builtbyjb.qrgen.config.PostgresConfig;
import com.builtbyjb.qrgen.model.QRCodeModel;
import io.github.cdimascio.dotenv.Dotenv;
import java.sql.Connection;
import java.sql.PreparedStatement;
// import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.List;
import java.util.Optional;

public class QRCodeRepository {

    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();
    private final String INSERT_SQL;
    private final String QR_CODE_TABLE_NAME;

    public QRCodeRepository() {
        QR_CODE_TABLE_NAME = Optional.ofNullable(System.getenv("QR_CODE_TABLE_NAME")).orElse(
            dotenv.get("QR_CODE_TABLE_NAME")
        );

        if (QR_CODE_TABLE_NAME == null || QR_CODE_TABLE_NAME.isEmpty()) {
            throw new IllegalArgumentException("ENV variable QR_CODE_TABLE_NAME not found");
        }

        INSERT_SQL = String.format(
            "INSERT INTO %s (code_value, project_id, location, status, processed, distributor_id) " + "VALUES (?, ?, ?, ?, ?, ?)",
            QR_CODE_TABLE_NAME
        );
    }

    public boolean insert(QRCodeModel qrCode) {
        try (
            Connection db = new PostgresConfig().getConnection();
            PreparedStatement stmt = db.prepareStatement(INSERT_SQL)
        ) {
            setStmtValues(stmt, qrCode);
            stmt.executeUpdate();
            db.commit();
            return true;
        } catch (SQLException e) {
            throw new RuntimeException("Failed to insert QR code", e);
        }
    }

    public boolean batchInsert(List<QRCodeModel> qrCodes) {
        try (
            Connection db = new PostgresConfig().getConnection();
            PreparedStatement stmt = db.prepareStatement(INSERT_SQL)
        ) {
            db.setAutoCommit(false);

            for (QRCodeModel qrCode : qrCodes) {
                setStmtValues(stmt, qrCode);
                stmt.addBatch();
            }

            stmt.executeBatch();
            stmt.clearBatch();
            db.commit();
            return true;
        } catch (SQLException e) {
            throw new RuntimeException("Failed to batch insert QR codes: " + e.getMessage());
        }
    }

    private void setStmtValues(PreparedStatement stmt, QRCodeModel qrCode) throws SQLException {
        stmt.setString(1, qrCode.getQRCode());
        stmt.setLong(2, qrCode.getProjectId());
        stmt.setString(3, qrCode.getLocation());
        stmt.setString(4, qrCode.getStatus());
        stmt.setBoolean(5, qrCode.getProcessed());
        stmt.setLong(6, qrCode.getDistributorId());
    }
}
