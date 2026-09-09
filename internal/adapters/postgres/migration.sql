CREATE TABLE IF NOT EXISTS dinapay_routing_decisions(request_id UUID PRIMARY KEY,request_hash TEXT NOT NULL,response JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now());
