-- +goose Up
CREATE TABLE campaigns (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'draft',
    enrolled_count INT  NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE enrollments (
    customer_id TEXT NOT NULL REFERENCES customers(id),
    campaign_id TEXT NOT NULL REFERENCES campaigns(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (customer_id, campaign_id)
);

-- +goose Down
DROP TABLE enrollments;
DROP TABLE campaigns;
