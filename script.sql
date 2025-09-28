-- Posts Domain Schema
-- Consolidated SQL script for all database setup

-- Enable PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Create enum types for type safety
CREATE TYPE post_status AS ENUM ('active', 'resolved', 'expired', 'deleted');
CREATE TYPE post_type AS ENUM ('lost', 'found');

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

-- Add comments for documentation
COMMENT ON TABLE posts IS 'Lost and found posts with geospatial location data';
COMMENT ON COLUMN posts.location IS 'PostGIS point geometry in WGS84 (SRID 4326) coordinate system';
COMMENT ON COLUMN posts.radius_meters IS 'Search radius in meters for this post (100m to 50km)';
COMMENT ON TABLE post_photos IS 'Photos associated with posts, supports 1-10 photos per post';
COMMENT ON COLUMN post_photos.display_order IS 'Display order of photos (1-10), unique per post';

-- Insert some sample data for testing (optional, can be removed in production)
-- Sample organization
INSERT INTO posts (title, description, location, type, user_id, organization_id) VALUES
('Lost iPhone 14', 'Black iPhone 14 Pro lost near Central Park', ST_SetSRID(ST_MakePoint(-73.9665, 40.7831), 4326), 'lost', gen_random_uuid(), gen_random_uuid()),
('Found Keys', 'Set of house keys found on 5th Avenue', ST_SetSRID(ST_MakePoint(-73.9599, 40.7736), 4326), 'found', gen_random_uuid(), gen_random_uuid());