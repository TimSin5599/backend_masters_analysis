from fastapi import FastAPI, UploadFile, File, Form
from pydantic import BaseModel
from typing import List, Optional
from engine import ExtractionEngine
from config import APP_TITLE, APP_DESCRIPTION, APP_VERSION, HTTP_PORT
import os
import shutil
import uvicorn
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title=APP_TITLE,
    description=APP_DESCRIPTION,
    version=APP_VERSION,
    docs_url="/docs",
    redoc_url="/redoc"
)
engine = ExtractionEngine()

UPLOAD_DIR = "temp_uploads"
os.makedirs(UPLOAD_DIR, exist_ok=True)

@app.post(
    "/extract",
    summary="Extract Data from Document",
    description="Upload a document (PDF or image) and specify a category to extract structured information using a Vision AI model."
)
async def extract_data(file: UploadFile = File(...), category: str = Form(...)):
    # 1. Save file locally
    temp_path = os.path.join(UPLOAD_DIR, file.filename)
    logger.info(f"Received file: {file.filename} for category: {category}")
    
    with open(temp_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)
    
    try:
        # 2. Extract structured Markdown text and context images using Docling
        images_b64, warnings = engine.extract_document_features(temp_path)
        
        if not images_b64:
            return {"error": "Failed to process document pages or no content found", "warnings": warnings}

        # 3. AI Processing with Qwen3-VL (Hybrid Markdown + Images)
        result = engine.process_with_ai(images_b64, category)
        
        if isinstance(result, dict) and "warnings" not in result:
            result["warnings"] = warnings
            
        return result
    except Exception as e:
        logger.error(f"Error in extract_data: {e}")
        return {"error": str(e)}
    finally:
        # 4. Cleanup
        if os.path.exists(temp_path):
            os.remove(temp_path)

@app.post(
    "/v1/classify",
    summary="Classify Document",
    description="Upload a document and determine its category using a Vision AI model."
)
async def classify_document(file: UploadFile = File(...)):
    temp_path = os.path.join(UPLOAD_DIR, file.filename)
    logger.info(f"Received file for classification: {file.filename}")
    
    with open(temp_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)
    
    try:
        images_b64, warnings = engine.extract_document_features(temp_path)
        if not images_b64:
            return {"category": "unknown", "warnings": warnings, "error": "Failed to process document"}
            
        return engine.classify_document(images_b64, warnings)
    except Exception as e:
        logger.error(f"Error in classify_document endpoint: {e}")
        return {"category": "unknown", "warnings": [f"Системная ошибка: {e}"], "error": str(e)}
    finally:
        if os.path.exists(temp_path):
            os.remove(temp_path)

@app.post(
    "/v1/classify/directory",
    summary="Classify Directory",
    description="Scan a local directory and classify all documents within it."
)
async def classify_directory(req: dict):
    dir_path = req.get("dirPath")
    if not dir_path:
        return {"error": "dirPath is required"}, 400
        
    try:
        results = engine.scan_directory_recursive(dir_path)
        total = len(results)
        classified = sum(1 for r in results if r['category'] != 'unknown')
        unknown = total - classified
        
        return {
            "results": results,
            "total": total,
            "classified": classified,
            "unknown": unknown
        }
    except Exception as e:
        logger.error(f"Error in classify_directory endpoint: {e}")
        return {"error": str(e)}, 500

class CriterionRequest(BaseModel):
    code: str
    title: str
    max_score: int
    scheme: Optional[str] = "default"


class ScoreRequest(BaseModel):
    criteria: List[CriterionRequest]
    applicant_data: dict


@app.post(
    "/v1/score",
    summary="Score Applicant Portfolio",
    description="Score applicant portfolio categories using AI based on evaluation criteria and extracted data."
)
async def score_portfolio(req: ScoreRequest):
    try:
        criteria = [c.model_dump() for c in req.criteria]
        results = engine.score_portfolio(criteria, req.applicant_data)
        return {"scores": results}
    except Exception as e:
        logger.error(f"Error in score_portfolio: {e}")
        return {"scores": [], "error": str(e)}


class AnnotationRequest(BaseModel):
    applicant_data: dict


@app.post(
    "/v1/annotate",
    summary="Generate Applicant Annotation",
    description="Generate a narrative annotation for an applicant based on all extracted data using AI."
)
async def generate_annotation(req: AnnotationRequest):
    try:
        annotation = engine.generate_annotation(req.applicant_data)
        return {"annotation": annotation}
    except Exception as e:
        logger.error(f"Error in generate_annotation: {e}")
        return {"annotation": "", "error": str(e)}


if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=HTTP_PORT, reload=True)
