package com.pesira.traceability.model;

public class QRCodeModel {

    private String qrCode;
    private Long projectId;
    private String location;
    private String status;
    private boolean processed;
    private Long distributorId;


    public QRCodeModel(String qrCode, Long projectId, String location, Long distributorId) {
        this.qrCode = qrCode;
        this.projectId = projectId;
        this.location = location;
        this.status = "AVAILABLE";
        this.processed = false;
        this.distributorId = distributorId;
    }

    public String getQRCode() {
        return qrCode;
    }

    public Long getProjectId() {
        return projectId;
    }

    public String getLocation() {
        return location;
    }

    public String getStatus() {
        return status;
    }

    public boolean getProcessed() {
        return processed;
    }

    public Long getDistributorId() {
        return distributorId;
    }

    @Override
    public String toString() {
        return "QRCodeModel{" +
                "qrCode='" + qrCode + '\'' +
                ", projectId='" + projectId + '\'' +
                ", location='" + location + '\'' +
                ", status='" + status + '\'' +
                ", processed=" + processed +
                '}';
    }
}
