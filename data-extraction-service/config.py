from dotenv import load_dotenv
import os

load_dotenv()

OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://host.docker.internal:11434")
MODEL_NAME = os.getenv("MODEL_NAME", "qwen3-vl:8b")
