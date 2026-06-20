-- MailerCloud Schema Initialization
-- Creates the events and campaign_stats tables

CREATE TABLE IF NOT EXISTS events (
    event_id    VARCHAR(64)  NOT NULL,
    campaign_id VARCHAR(64)  NOT NULL,
    type        ENUM('sent', 'opened', 'clicked', 'bounced') NOT NULL,
    timestamp   DATETIME(3)  NOT NULL,
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (event_id),
    INDEX idx_events_campaign_type (campaign_id, type)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  ROW_FORMAT=COMPRESSED;

CREATE TABLE IF NOT EXISTS campaign_stats (
    campaign_id   VARCHAR(64)  NOT NULL,
    sent_count    INT UNSIGNED NOT NULL DEFAULT 0,
    opened_count  INT UNSIGNED NOT NULL DEFAULT 0,
    clicked_count INT UNSIGNED NOT NULL DEFAULT 0,
    bounced_count INT UNSIGNED NOT NULL DEFAULT 0,
    updated_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (campaign_id)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4;
