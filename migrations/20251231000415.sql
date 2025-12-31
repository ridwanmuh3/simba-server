-- Create "items" table
CREATE TABLE "public"."items" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "item_code" character varying NULL,
  "name" character varying NULL,
  "category" character varying NULL,
  "quantity" bigint NULL,
  "unit_price" numeric NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_items_deleted_at" to table: "items"
CREATE INDEX "idx_items_deleted_at" ON "public"."items" ("deleted_at");
-- Create index "idx_items_item_code" to table: "items"
CREATE UNIQUE INDEX "idx_items_item_code" ON "public"."items" ("item_code", "item_code");
-- Create "users" table
CREATE TABLE "public"."users" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "email" character varying NULL,
  "fullname" character varying NULL,
  "role" character varying NULL,
  "password" character varying NULL,
  "token" character varying NULL,
  "is_active" boolean NULL DEFAULT false,
  "last_active" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "public"."users" ("deleted_at");
-- Create index "idx_users_token" to table: "users"
CREATE INDEX "idx_users_token" ON "public"."users" ("token");
-- Create index "idx_users_username" to table: "users"
CREATE UNIQUE INDEX "idx_users_username" ON "public"."users" ("email");
