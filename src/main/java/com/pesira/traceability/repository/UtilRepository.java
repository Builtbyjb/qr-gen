package com.pesira.traceability.repository;

import com.google.cloud.datastore.*;
import io.github.cdimascio.dotenv.Dotenv;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

public class UtilRepository {

    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();
    private final Datastore datastore;
    private final String ENTITY_ID;
    private final String ENTITY_KIND;

    public UtilRepository() {
        ENTITY_ID = Optional.ofNullable(System.getenv("ENTITY_ID")).orElse(dotenv.get("ENTITY_ID"));
        if (ENTITY_ID == null || ENTITY_ID.isEmpty()) throw new IllegalArgumentException(
            "ENV variable ENTITY_ID not found"
        );

        ENTITY_KIND = Optional.ofNullable(System.getenv("ENTITY_KIND")).orElse(dotenv.get("ENTITY_KIND"));
        if (ENTITY_KIND == null || ENTITY_KIND.isEmpty()) throw new IllegalArgumentException(
            "ENV variable ENTITY_KIND not found"
        );

        this.datastore = DatastoreOptions.getDefaultInstance().getService();
    }

    public long getCursorEntity() {
        KeyFactory keyFactory = datastore.newKeyFactory().setKind(ENTITY_KIND).setNamespace("util");
        Key key = keyFactory.newKey(Long.parseLong(ENTITY_ID));
        Entity entity = datastore.get(key);
        return entity.getLong("cursor");
    }

    public void updateCursorEntity(long cursor) {
        KeyFactory keyFactory = datastore.newKeyFactory().setKind(ENTITY_KIND).setNamespace("util");
        Key key = keyFactory.newKey(Long.parseLong(ENTITY_ID));
        Entity entity = datastore.get(key);

        if (entity != null) {
            Entity updatedEntity = Entity.newBuilder(entity).set("cursor", cursor).build();
            datastore.update(updatedEntity);
        }
    }

    public Map<String, String> getTokenEntities() {
        Map<String, String> tokens = new HashMap<>();
        KeyFactory keyFactory = datastore.newKeyFactory().setKind(ENTITY_KIND).setNamespace("util");

        // Get access token
        Key accessKey = keyFactory.newKey(Long.parseLong(ENTITY_ID));
        Entity accessEntity = datastore.get(accessKey);
        tokens.put("accessToken", accessEntity.getString("access_token"));

        // Get refresh token
        Key refreshKey = keyFactory.newKey(Long.parseLong(ENTITY_ID));
        Entity refreshEntity = datastore.get(refreshKey);
        tokens.put("refreshToken", refreshEntity.getString("refresh_token"));

        return tokens;
    }

    public void updateTokenEntities(String accessToken, String refreshToken) {
        KeyFactory keyFactory = datastore.newKeyFactory().setKind(ENTITY_KIND).setNamespace("util");

        // Update access token
        Key accessKey = keyFactory.newKey(Long.parseLong(ENTITY_ID));
        Entity accessEntity = datastore.get(accessKey);

        if (accessEntity != null) {
            Entity updateAccessEntity = Entity.newBuilder(accessEntity).set("access_token", accessToken).build();
            datastore.update(updateAccessEntity);
        }

        // Update refresh token
        Key refreshKey = keyFactory.newKey(Long.parseLong(ENTITY_ID));
        Entity refreshEntity = datastore.get(refreshKey);

        if (refreshEntity != null) {
            Entity updateRefreshEntity = Entity.newBuilder(refreshEntity).set("refresh_token", refreshToken).build();
            datastore.update(updateRefreshEntity);
        }
    }
}
