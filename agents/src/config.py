"""Configuration management for NeoGraph agents."""
import os
from typing import Optional
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Environment configuration for agents service."""

    # Anthropic API - supports both API key and auth token methods
    anthropic_api_key: str = os.getenv("ANTHROPIC_API_KEY", "")
    anthropic_auth_token: Optional[str] = os.getenv("ANTHROPIC_AUTH_TOKEN")
    anthropic_base_url: Optional[str] = os.getenv("ANTHROPIC_BASE_URL")

    # Neo4j connection
    neo4j_uri: str = os.getenv("NEO4J_URI", "bolt://localhost:7687")
    neo4j_user: str = os.getenv("NEO4J_USER", "neo4j")
    neo4j_password: str = os.getenv("NEO4J_PASSWORD", "password")

    # Service configuration
    host: str = "0.0.0.0"
    port: int = 8001

    # CORS
    cors_origins: list[str] = ["*"]

    class Config:
        env_file = ".env"


settings = Settings()
