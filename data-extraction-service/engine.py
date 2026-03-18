import ollama
import fitz  # PyMuPDF
import base64
from PIL import Image
import io
import logging
import json
import re
import os

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ExtractionEngine:
    def __init__(self):
        # We use qwen3-vl:4b for state-of-the-art vision extraction
        self.client = ollama.Client(host=os.getenv("OLLAMA_HOST", "http://host.docker.internal:11434"), timeout=1800)
        self.model_name = "qwen3-vl:8b"
        logger.info(f"ExtractionEngine initialized ({self.model_name}) natively")

    def extract_document_features(self, file_path: str) -> list[str]:
        """Extracts text and embedded images natively without OCR."""
        logger.info(f"Extracting features from file: {file_path}")
        images_base64 = []
        
        try:
            if file_path.lower().endswith('.pdf'):
                doc = fitz.open(file_path)
                for page_num, page in enumerate(doc):
                    # Render full page as image (like a screenshot) at 2x resolution
                    try:
                        zoom = 1.5  # Lower zoom to prevent context limit overflow while maintaining readability
                        mat = fitz.Matrix(zoom, zoom)
                        pix = page.get_pixmap(matrix=mat)
                        
                        # Convert pixmap to PIL Image then to JPEG
                        pil_img = Image.frombytes("RGB", [pix.width, pix.height], pix.samples)
                        out_buffered = io.BytesIO()
                        pil_img.save(out_buffered, format="JPEG", quality=85)
                        
                        # # Save debug image to inspect quality
                        # debug_dir = "/app/debug_images"
                        # os.makedirs(debug_dir, exist_ok=True)
                        # debug_path = os.path.join(debug_dir, f"page_{page_num}.jpg")
                        # pil_img.save(debug_path, format="JPEG", quality=85)
                        # logger.info(f"Saved debug image: {debug_path}")
                        
                        img_str = base64.b64encode(out_buffered.getvalue()).decode("utf-8")
                        images_base64.append(img_str)
                        logger.info(f"Rendered page {page_num} as image ({pix.width}x{pix.height})")
                    except Exception as e:
                        logger.warning(f"Could not render page {page_num} as image: {e}")
                        
                logger.info(f"Extracted {len(images_base64)} page images from PDF.")
                
            elif file_path.lower().endswith(('.png', '.jpg', '.jpeg')):
                with open(file_path, "rb") as f:
                    img_data = f.read()
                    
                    # Normalize image to JPEG
                    buffered = io.BytesIO(img_data)
                    pil_img = Image.open(buffered)
                    if pil_img.mode in ('RGBA', 'P'):
                        pil_img = pil_img.convert('RGB')
                    out_buffered = io.BytesIO()
                    pil_img.save(out_buffered, format="JPEG")
                    
                    images_base64.append(base64.b64encode(out_buffered.getvalue()).decode('utf-8'))
                logger.info("Extracted 1 regular image natively.")
            
            return images_base64
        except Exception as e:
            logger.error(f"Failed to extract document features natively: {e}", exc_info=True)
            return images_base64

    def process_with_ai(self,images_b64: list[str], category: str) -> dict:
        """Sends context Photos to Ollama (Qwen3-VL) for structured data extraction."""
        prompts = {
            "passport": (
                "Extract passport data into a JSON object. IMPORTANT: Use exactly these keys: "
                "\"surname\" (look for 'Surname' - can consist of multiple words), \"name\" (look for 'Given Names' - can consist of multiple words), \"patronymic\" (look for 'Father Name'), "
                "\"document_number\" (serial number), \"date_of_birth\" (YYYY-MM-DD), \"nationality\", \"gender\" (M or F). "
                "If 'Given Names' contains multiple words, extract them all into 'name'. "
                "If 'Surname' contains multiple words, extract them all into 'surname'. "
                "If 'Father Name' is present, map it to 'patronymic'. "
                "Ensure date_of_birth is strictly YYYY-MM-DD. "
                "Output ONLY the JSON object."
            ),
            "diploma": (
                "Extract diploma details into a JSON object with exactly these keys: "
                "institution_name, degree_title (must be only 'bachelor', 'master', or similar standard degree tier), major, graduation_date (YYYY-MM-DD format), diploma_serial_number. "
                "Look for registration or serial numbers. "
                "Output ONLY the raw JSON object. Do not explain your thought process."
            ),
            "transcript": (
                "Extract cumulative academic transcript data from the document. "
                "HINT: These final cumulative values are usually located at the end of the document. "
                "Return a JSON object with exactly the following schema keys: "
                "{\"cumulative_gpa\": number, \"cumulative_grade\": string, \"total_credits\": number, \"obtained_credits\": number, \"total_semesters\": number} "
                "cumulative_gpa: the final numeric GPA or average score. "
                "cumulative_grade: the final letter grade or text (e.g., 'A', 'A+', 'B', 'Excellent'). Do NOT output a float/number here. "
                "total_credits: look for 'Maximum Marks' or 'Max Credits' at the end of the document. "
                "obtained_credits: look for 'Obtained Marks', 'Earned Credits', or similar at the end of the document. "
                "total_semesters: the highest or last semester number shown on the transcript (e.g., if lists semesters 1 through 8, out is 8). "
                "Do NOT extract individual subjects or grades. Ignore any subject lists. "
                "Missing fields: use \"\" for missing strings and null for missing numbers. "
                "Output ONLY the JSON object."
            ),
            "motivation": (
                "Analyze this motivation letter. Extract as JSON with one key: 'main_text'. "
                "main_text should contain the core content of the letter — the key thoughts, achievements, reasons for applying — "
                "stripped of any generic/filler/introductory phrases. Keep it concise but informative. Write in the same language as the original."
            ),
            "recommendation": (
                "Analyze recommendation letter for JSON: author_name (who wrote the letter), "
                "author_position (their title/role), author_institution (their organization), "
                "key_strengths (what qualities they highlight about the candidate)."
            ),
            "resume": (
                "Extract specific data from CV/Resume. Return a JSON object with strictly these keys: "
                "'personal_data': {'email': string, 'phone': string}, "
                "'experiences': list of objects with {'company_name': string, 'position': string, 'start_date': YYYY-MM-DD, 'end_date': YYYY-MM-DD or null, 'record_type': 'work' or 'internship' or 'training'}, "
                "'achievements': list of objects with {'achievement_type': string, 'achievement_title': string, 'company_name': string, 'date_received': YYYY-MM-DD}. "
                "CRITICAL: Be extremely consistent with company names and job titles. Do not include 'LLC', 'Inc', or extra descriptors unless they are part of the core brand. "
                "For 'phone', if multiple numbers are found, separate them with a comma and space (e.g., '+123, +456'). "
                "Ensure dates are strictly YYYY-MM-DD. If only a year or month/year is provided, use the first day of that period (e.g., '2023' -> '2023-01-01')."
                "Extract ALL work experiences and ALL significant achievements."
            ),
            "achievement": (
                "Extract achievement/certificate details: achievement_type (e.g., certificate, diploma, award, letter), "
                "achievement_title (name of the award/certificate), company (issuing organization), "
                "date_received (in YYYY-MM-DD format)."
            ),
            "language": (
                "Extract English language certificate data: exam_name (e.g. IELTS, TOEFL, Cambridge, etc), "
                "score (the total points/band/grade), english_level (e.g. B2, C1, etc if mentioned). "
                "This document is specifically an English language proficiency certificate."
            ),
            "work": (
                "Extract career/work history. Return a JSON with one key 'experiences' which is a list of objects. "
                "Each object must contain: company_name, position, start_date (YYYY-MM-DD), end_date (YYYY-MM-DD or empty if current job). "
                "Extract ALL jobs/positions mentioned in the document."
            ),
            "prof_development": (
                "Extract career/work history or training details. Return a JSON with one key 'experiences' which is a list of objects. "
                "Each object must contain: company_name, position, start_date (YYYY-MM-DD), end_date (YYYY-MM-DD or empty if current job). "
                "Extract ALL jobs, internships or training programs mentioned in the document."
            ),
            "second_diploma": (
                "Extract education data from a diploma. Return a JSON with: institution_name, degree_title, major, "
                "graduation_date (YYYY-MM-DD), diploma_serial_number."
            ),
            "certification": (
                "Extract achievement/certificate details: achievement_type (e.g., certificate, diploma, award, letter), "
                "achievement_title (name of the award/certificate), company (issuing organization), "
                "date_received (in YYYY-MM-DD format)."
            )
        }
        
        system_prompt = (
            """You are a data extraction assistant. Follow these rules:
            1. Always output exactly one JSON object and nothing else.
            2. Use only the specified field names. Do not change their order or casing.
            3. Do not include any explanations, text, or markdown—only the raw JSON (no code fences).
            4. If a value is missing, use null or an empty string "". If you are not sure and must guess, prefix the value with '~'.
            5. Ensure the JSON is valid (keys in double quotes, etc.).
            6. DO NOT use <think> tags or output reasoning steps. Output the JSON block IMMEDIATELY."""
        )
        
        # Handle category suffixes (e.g., 'prof_development:internship')
        base_category = category.split(':')[0] if ':' in category else category
        user_task = prompts.get(category) or prompts.get(base_category)
        logger.info(f"Processing AI extraction for category: {category} (base: {base_category}). Task found: {user_task is not None}")
        
        if not user_task:
            logger.warning(f"No specific prompt found for category: {category}. Falling back to default.")
            user_task = "Extract data as JSON."

        prompt = f"Category: {category}\n\nTask: {user_task}"
        logger.info(f"Prompt prepared. Length: {len(prompt)}")
        
        max_retries = 3
        last_error = "Unknown error"
        
        for attempt in range(max_retries):
            try:
                logger.info(f"Ollama chat started for {category} [Attempt {attempt+1}]")
                response = self.client.chat(
                    model=self.model_name,
                    messages=[
                        {'role': 'system', 'content': system_prompt},
                        {
                            'role': 'user', 
                            'content': prompt,
                            'images': images_b64
                        },
                    ],
                    options={
                        "num_ctx": 32768, # Required for multi-page documents like transcripts
                        "temperature": 0.0
                    }
                )
                logger.info(f"Ollama chat completed for {category} [Attempt {attempt+1}]")
                
                # Debug: log full response metadata
                resp_dict = response.model_dump() if hasattr(response, 'model_dump') else dict(response)
                logger.info(f"Ollama done_reason: {resp_dict.get('done_reason', 'N/A')}")
                dur = resp_dict.get('total_duration')
                if dur:
                    logger.info(f"Total duration: {dur / 1e9:.1f}s")
                # Clean and parse JSON
                data, clean_content = self._parse_json_response(response.message.content)
                logger.info(f"AI Final JSON [Attempt {attempt+1}]: {clean_content[:300]}...")
                
                if "error" not in data:
                    if not data: # Check if dictionary is empty {}
                        last_error = "AI returned an empty JSON object {}"
                        logger.warning(f"Failed to extract valid data on attempt {attempt+1}: {last_error}. Retrying...")
                        continue
                        
                    # Normalize and sanitize data
                    return self._normalize_and_sanitize(data)
                
                last_error = data.get("error", "Parse error")
                logger.warning(f"Failed to extract valid data on attempt {attempt+1}: {last_error}. Retrying...")
                # Tell the AI to try again and fix its JSON next iteration implicitly by just retrying

            except Exception as e:
                last_error = str(e)
                logger.error(f"Error calling AI on attempt {attempt+1}: {e}", exc_info=True)
                
        return {"error": "AI processing failed after retries", "details": last_error}

    def _parse_json_response(self, content: str) -> tuple[dict, str]:
        """Cleans up AI response and robustly extracts JSON."""
        # Strip <think>...</think> tags that qwen3-vl models may produce
        logger.info(f"Ollama content: {content}")
        clean = re.sub(r'<think>.*?</think>', '', content, flags=re.DOTALL)
        
        # Identify if there is an unclosed <think> tag
        last_think_idx = clean.rfind('<think>')
        if last_think_idx != -1 and '</think>' not in clean[last_think_idx:]:
            logger.warning("Unclosed <think> tag detected! Removing the unclosed think block.")
            clean = clean[:last_think_idx]

        clean = clean.strip()
        
        if not clean:
            logger.warning("AI returned COMPLETELY EMPTY response content (after stripping thinking tags).")
            return {"error": "Empty response from AI"}, ""

        # Basic cleaning
        clean = clean.strip()
        
        # Remove markdown/comments
        clean = re.sub(r'^```(?:json)?\n?', '', clean, flags=re.MULTILINE)
        clean = re.sub(r'\n?```$', '', clean, flags=re.MULTILINE)
        clean = re.sub(r'//.*', '', clean)
        
        # Try to find JSON block using regex if there's garbage text around it
        json_match = re.search(r'(\{.*\})', clean, re.DOTALL)
        if json_match:
            clean = json_match.group(1)
        
        try:
            data = json.loads(clean)
            if len(data) == 1 and isinstance(list(data.values())[0], dict) and list(data.keys())[0] not in ["subjects", "skills"]:
                 data = list(data.values())[0]
            return data, clean
        except Exception as e:
            logger.error(f"JSON parsing failed: {e}. Cleaned content: {clean[:200]}")
            return {"error": "JSON parse error", "raw": clean}, clean

    def _normalize_and_sanitize(self, data: dict) -> dict:
        """Normalizes keys and ensures string values for the backend."""
        normalized = {}
        for k, v in data.items():
            low_k = k.lower().replace(" ", "_").replace("-", "_")
            
            # Robust Mapping
            if low_k in ["cgpa", "overall_gpa", "gpa", "cumulative_gpa"]:
                normalized["cumulative_gpa"] = v
            elif low_k in ["maximum_marks", "total_marks", "max_credits", "total_credits"]:
                normalized["total_credits"] = v
            elif low_k in ["obtained_marks", "earned_credits", "obtained_credits", "marks_obtained"]:
                normalized["obtained_credits"] = v
            elif low_k in ["overall_grade", "grade", "cumulative_grade"]:
                # Ensure it's not a numeric string masquerading as a grade
                if v and str(v).replace('.', '', 1).isdigit():
                    logger.warning(f"Rejecting numeric value {v} for cumulative_grade")
                    normalized["cumulative_grade"] = ""
                else:
                    normalized["cumulative_grade"] = v
            elif "semester" in low_k and "gpa" in low_k:
                match = re.search(r'\d+', low_k)
                if match: normalized[f"gpa_semester_{match.group()}"] = v
                else: normalized[k] = v
            elif low_k in ["passport_number", "serial_number", "id_number", "document_number"]:
                normalized["document_number"] = v
            elif low_k in ["sex", "gender"]:
                # Normalize to M/F or empty
                val = str(v).upper()
                if "M" in val or "М" in val: normalized["gender"] = "M"
                elif "F" in val or "W" in val or "Ж" in val or "F" in val: normalized["gender"] = "F"
                else: normalized["gender"] = v
            elif low_k in ["given_names", "first_name"]:
                normalized["name"] = v
            elif low_k in ["last_name", "surname"]:
                normalized["surname"] = v
            elif low_k in ["father_name", "father", "patronymic"]:
                normalized["patronymic"] = v
            elif low_k == "name" and "surname" not in data and "last_name" not in data and "given_names" not in data:
                # If only 'name' is provided, it might be the full name
                parts = str(v).split()
                if len(parts) >= 2:
                    normalized["surname"] = parts[0]
                    normalized["name"] = parts[1]
                    if len(parts) > 2:
                        normalized["patronymic"] = " ".join(parts[2:])
                else:
                    normalized["name"] = v
            else:
                normalized[k] = v

        # Flatten personal_data if it exists
        if "personal_data" in normalized and isinstance(normalized["personal_data"], dict):
            pd = normalized.pop("personal_data")
            if "email" in pd and pd["email"]:
                normalized["email"] = pd["email"]
            if "phone" in pd and pd["phone"]:
                normalized["phone"] = pd["phone"]

        # Sanitize for Go backend (needs map[string]string mostly, except 'subjects', 'experiences', 'achievements')
        sanitized = {}
        for k, v in normalized.items():
            if k in ["subjects", "experiences", "achievements"] and isinstance(v, list):
                sanitized[k] = json.dumps(v)
            elif isinstance(v, list):
                if v and isinstance(v[0], dict):
                    sanitized[k] = "; ".join([", ".join([str(val) for val in item.values()]) for item in v])
                else:
                    sanitized[k] = ", ".join([str(item) for item in v])
            elif v is None:
                sanitized[k] = ""
            else:
                if k == "cumulative_gpa":
                    sanitized[k] = str(v).replace(",", ".")
                else:
                    sanitized[k] = str(v)
        
        return sanitized
