from fastapi import FastAPI, UploadFile, File, Form
from engine import ExtractionEngine
import os
import shutil
import uvicorn
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Data Extraction Service",
    description="Service for extracting structured data from documents (PDFs, images) using Vision AI models.",
    version="1.0.0",
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
        images_b64 = engine.extract_document_features(temp_path)
        
        if not images_b64:
            return {"error": "Failed to process document pages or no content found"}

        # 3. AI Processing with Qwen3-VL (Hybrid Markdown + Images)
        result = engine.process_with_ai(images_b64, category)
        
        return result
    except Exception as e:
        logger.error(f"Error in extract_data: {e}")
        return {"error": str(e)}
    finally:
        # 4. Cleanup
        if os.path.exists(temp_path):
            os.remove(temp_path)

if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True)
