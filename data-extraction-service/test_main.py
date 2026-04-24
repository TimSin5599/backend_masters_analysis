import pytest
from fastapi.testclient import TestClient
from main import app
import os
import shutil

# Mocking the engine methods to avoid actual AI processing
@pytest.fixture
def mock_engine(monkeypatch):
    class MockEngine:
        def __init__(self):
            pass
        
        def extract_document_features(self, temp_path):
            return ["base64_fake_image"], []
        
        def process_with_ai(self, images_b64, category):
            return {"fake_data": "some_value"}
        
        def classify_document(self, images_b64, warnings):
            return {"category": "diploma", "warnings": []}
            
        def scan_directory_recursive(self, dir_path):
            return [{"category": "diploma", "fileName": "test.pdf", "error": None}]

        def score_portfolio(self, criteria, applicant_data):
            return [{"code": "BASE", "score": 10, "comment": "Good"}]

    monkeypatch.setattr("main.engine", MockEngine())

@pytest.fixture
def client(mock_engine):
    return TestClient(app)

def test_classify_directory(client):
    response = client.post("/v1/classify/directory", json={"dirPath": "/tmp"})
    assert response.status_code == 200
    data = response.json()
    assert data["total"] == 1
    assert data["classified"] == 1
    assert data["results"][0]["category"] == "diploma"

def test_score_portfolio(client):
    payload = {
        "criteria": [
            {"code": "BASE", "title": "Base", "max_score": 10}
        ],
        "applicant_data": {
            "some_key": "some_value"
        }
    }
    response = client.post("/v1/score", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert len(data["scores"]) == 1
    assert data["scores"][0]["score"] == 10
