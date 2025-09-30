-- Posts Domain Schema
-- Consolidated SQL script for all database setup

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Create enum types for type safety
CREATE TYPE post_status AS ENUM ('active', 'resolved', 'expired', 'deleted');
CREATE TYPE post_type AS ENUM ('lost', 'found');
CREATE TYPE contact_exchange_status AS ENUM ('pending', 'approved', 'denied', 'expired');
CREATE TYPE contact_exchange_approval_type AS ENUM ('full_contact', 'platform_message', 'limited_contact');
CREATE TYPE verification_method AS ENUM ('photo_proof', 'security_question', 'admin_approval');
CREATE TYPE denial_reason AS ENUM ('not_owner', 'insufficient_verification', 'suspicious_request', 'post_resolved', 'user_preference', 'other');

-- Create posts table
CREATE TABLE posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    location        GEOMETRY(POINT, 4326),  -- PostGIS point with WGS84 coordinate system
    radius_meters   INTEGER DEFAULT 1000 CHECK (radius_meters >= 100 AND radius_meters <= 50000),
    status          post_status DEFAULT 'active',
    type           post_type NOT NULL,
    user_id        UUID NOT NULL,
    organization_id UUID,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT posts_title_not_empty CHECK (length(trim(title)) > 0)
);

-- Create post_photos table for 1-to-many relationship
CREATE TABLE post_photos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    thumbnail_url TEXT,
    caption     TEXT,
    display_order INTEGER NOT NULL CHECK (display_order >= 1 AND display_order <= 10),
    format      VARCHAR(10) NOT NULL CHECK (format IN ('jpg', 'jpeg', 'png', 'webp')),
    size_bytes  BIGINT NOT NULL CHECK (size_bytes > 0),
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Ensure unique display order per post
    UNIQUE(post_id, display_order)
);

-- Create indexes for performance

-- Primary geospatial index for location-based queries
CREATE INDEX idx_posts_location_gist ON posts USING GIST (location);

-- Index for status and type filtering (most common queries)
CREATE INDEX idx_posts_status_type ON posts (status, type);

-- Index for multi-tenant queries
CREATE INDEX idx_posts_org_status ON posts (organization_id, status) WHERE organization_id IS NOT NULL;

-- Index for temporal queries (recent posts first)
CREATE INDEX idx_posts_created_at ON posts (created_at DESC);

-- Index for user posts lookup
CREATE INDEX idx_posts_user_id ON posts (user_id);

-- Index for post photos ordering
CREATE INDEX idx_post_photos_post_display ON post_photos (post_id, display_order);

-- Indexes for contact exchange requests
CREATE INDEX idx_contact_exchange_post_id ON contact_exchange_requests (post_id);
CREATE INDEX idx_contact_exchange_requester ON contact_exchange_requests (requester_user_id, status);
CREATE INDEX idx_contact_exchange_owner ON contact_exchange_requests (owner_user_id, status);
CREATE INDEX idx_contact_exchange_status ON contact_exchange_requests (status);
CREATE INDEX idx_contact_exchange_expires ON contact_exchange_requests (expires_at) WHERE status IN ('pending', 'approved');
CREATE INDEX idx_contact_exchange_created ON contact_exchange_requests (created_at DESC);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for posts table
CREATE TRIGGER update_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create trigger for contact_exchange_requests table
CREATE TRIGGER update_contact_exchange_updated_at
    BEFORE UPDATE ON contact_exchange_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create contact_exchange_requests table for secure contact sharing
CREATE TABLE contact_exchange_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id         UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    requester_user_id UUID NOT NULL,
    owner_user_id   UUID NOT NULL,
    status          contact_exchange_status DEFAULT 'pending',
    message         TEXT,
    verification_required BOOLEAN DEFAULT false,
    verification_method verification_method,
    verification_question TEXT,
    verification_requirements JSONB,
    approval_type   contact_exchange_approval_type,
    denial_reason   denial_reason,
    denial_message  TEXT,
    encrypted_contact_info JSONB,
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT contact_exchange_different_users CHECK (requester_user_id != owner_user_id),
    CONSTRAINT contact_exchange_expires_future CHECK (expires_at > created_at)
);

-- Add comments for documentation
COMMENT ON TABLE posts IS 'Lost and found posts with geospatial location data';
COMMENT ON COLUMN posts.location IS 'PostGIS point geometry in WGS84 (SRID 4326) coordinate system';
COMMENT ON COLUMN posts.radius_meters IS 'Search radius in meters for this post (100m to 50km)';
COMMENT ON TABLE post_photos IS 'Photos associated with posts, supports 1-10 photos per post';
COMMENT ON COLUMN post_photos.display_order IS 'Display order of photos (1-10), unique per post';
COMMENT ON TABLE contact_exchange_requests IS 'Secure contact exchange requests between post owners and interested users';
COMMENT ON COLUMN contact_exchange_requests.encrypted_contact_info IS 'Encrypted contact information (email/phone) when approved';
COMMENT ON COLUMN contact_exchange_requests.verification_requirements IS 'JSON array of verification requirements';

-- Create encryption_keys table for RSA-4096 key management
CREATE TABLE encryption_keys (
    id              VARCHAR(255) PRIMARY KEY,
    fingerprint     VARCHAR(255) UNIQUE NOT NULL,
    private_key     TEXT NOT NULL,  -- PEM encoded private key
    public_key      TEXT NOT NULL,  -- PEM encoded public key
    is_active       BOOLEAN DEFAULT false,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at      TIMESTAMP WITH TIME ZONE,

    -- Only one active key at a time
    CONSTRAINT one_active_key EXCLUDE (is_active WITH =) WHERE (is_active = true)
);

-- Create encryption_audit_logs table for compliance tracking
CREATE TABLE encryption_audit_logs (
    id              VARCHAR(255) PRIMARY KEY,
    operation       VARCHAR(50) NOT NULL CHECK (operation IN ('encrypt', 'decrypt', 'token_create', 'token_validate', 'key_rotation')),
    user_id         UUID NOT NULL,
    request_id      UUID,  -- Optional reference to contact exchange request
    key_fingerprint VARCHAR(255) NOT NULL,
    success         BOOLEAN NOT NULL,
    error_message   TEXT,
    timestamp       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address      INET,
    user_agent      TEXT
);

-- Indexes for encryption tables
CREATE INDEX idx_encryption_keys_active ON encryption_keys (is_active) WHERE is_active = true;
CREATE INDEX idx_encryption_keys_fingerprint ON encryption_keys (fingerprint);
CREATE INDEX idx_encryption_keys_created ON encryption_keys (created_at DESC);

CREATE INDEX idx_audit_logs_user_timestamp ON encryption_audit_logs (user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_request_timestamp ON encryption_audit_logs (request_id, timestamp DESC) WHERE request_id IS NOT NULL;
CREATE INDEX idx_audit_logs_operation_timestamp ON encryption_audit_logs (operation, timestamp DESC);
CREATE INDEX idx_audit_logs_success ON encryption_audit_logs (success, timestamp DESC);

-- Comments for encryption tables
COMMENT ON TABLE encryption_keys IS 'RSA-4096 encryption keys for secure contact token management';
COMMENT ON COLUMN encryption_keys.fingerprint IS 'SHA-256 fingerprint of the public key for identification';
COMMENT ON COLUMN encryption_keys.private_key IS 'PEM encoded RSA-4096 private key (encrypted storage recommended)';
COMMENT ON COLUMN encryption_keys.public_key IS 'PEM encoded RSA-4096 public key';
COMMENT ON COLUMN encryption_keys.is_active IS 'Only one key should be active at any time';

COMMENT ON TABLE encryption_audit_logs IS 'Audit trail for all encryption/decryption operations for compliance';
COMMENT ON COLUMN encryption_audit_logs.operation IS 'Type of encryption operation performed';
COMMENT ON COLUMN encryption_audit_logs.key_fingerprint IS 'Fingerprint of the encryption key used';
COMMENT ON COLUMN encryption_audit_logs.request_id IS 'Optional reference to contact exchange request';

-- Insert some sample data for testing (optional, can be removed in production)
-- Sample organization
INSERT INTO posts (title, description, location, type, user_id, organization_id) VALUES
('Lost iPhone 14', 'Black iPhone 14 Pro lost near Central Park', ST_SetSRID(ST_MakePoint(-73.9665, 40.7831), 4326), 'lost', gen_random_uuid(), gen_random_uuid()),
('Found Keys', 'Set of house keys found on 5th Avenue', ST_SetSRID(ST_MakePoint(-73.9599, 40.7736), 4326), 'found', gen_random_uuid(), gen_random_uuid());