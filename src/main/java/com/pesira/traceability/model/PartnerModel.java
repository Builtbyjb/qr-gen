package com.pesira.traceability.model;

public class PartnerModel {

    private String name;
    private int households;

    public PartnerModel(String name, int households) {
        this.name = name;
        this.households = households;
    }

    public String getName() {
        return name;
    }

    public int getHouseholds() {
        return households;
    }

    @Override
    public String toString() {
        return "PartnerModel{" + "name='" + name + '\'' + ", households=" + households + "}";
    }
}
