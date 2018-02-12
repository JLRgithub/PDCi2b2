CREATE TABLE i2b2demodata.demo_data_clear
(
    location_cd character varying(50) COLLATE pg_catalog."default",
    "time" character numeric(18,5) COLLATE pg_catalog."default",
    concept_path character varying(50) COLLATE pg_catalog."default",
    totalnum character varying(88) COLLATE pg_catalog."default"
)
WITH (
    OIDS = FALSE
)
TABLESPACE pg_default;

ALTER TABLE i2b2demodata.demo_data_clear
    OWNER to i2b2demodata;