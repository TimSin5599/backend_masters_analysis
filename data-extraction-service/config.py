from dotenv import load_dotenv
import os

load_dotenv()

OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://host.docker.internal:11434")
MODEL_NAME = os.getenv("MODEL_NAME", "qwen3-vl:8b")

APP_TITLE = os.getenv("APP_TITLE", "Data Extraction Service")
APP_DESCRIPTION = os.getenv("APP_DESCRIPTION", "Service for extracting structured data from documents using Vision AI models.")
APP_VERSION = os.getenv("APP_VERSION", "1.0.0")
HTTP_PORT = int(os.getenv("HTTP_PORT", "8000"))
