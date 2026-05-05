from prompts import classification_prompt, extraction_prompts, system_prompt
import ollama
import fitz  # PyMuPDF
import base64
from PIL import Image
import io
import logging
import json
import re
import os
import config

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ExtractionEngine:
    def __init__(self):
        
        self.client = ollama.Client(host=config.OLLAMA_HOST, timeout=300)
        self.model_name = config.MODEL_NAME
        self.classification_prompt = classification_prompt
        logger.info(f"ExtractionEngine initialized ({self.model_name}) natively")

    def extract_document_features(self, file_path: str) -> tuple[list[str], list[str]]:
        """Extracts text and embedded images natively without OCR. Returns (images_b64, warnings)."""
        logger.info(f"Extracting features from file: {file_path}")
        images_base64 = []
        warnings = []
        
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
                        warnings.append(f"Не удалось прочитать страницу {page_num + 1} документа.")
                        
                logger.info(f"Extracted {len(images_base64)} page images from PDF.")
                
            elif file_path.lower().endswith(('.png', '.jpg', '.jpeg')):
                try:
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
                except Exception as e:
                    logger.warning(f"Could not open image: {e}")
                    warnings.append(f"Не удалось открыть изображение.")
            
            return images_base64, warnings
        except Exception as e:
            logger.error(f"Failed to extract document features natively: {e}", exc_info=True)
            warnings.append(f"Не удалось открыть файл.")
            return images_base64, warnings

    def process_with_ai(self,images_b64: list[str], category: str) -> dict:
        """Sends context Photos to Ollama (Qwen3-VL) for structured data extraction."""
        prompts = extraction_prompts
        
        # For diploma: always run both diploma and transcript extractions and merge results.
        # Transcript fields will be empty if no grades table is present — that's fine.
        if category == "diploma":
            diploma_data = self._extract_single(images_b64, "diploma")
            transcript_data = self._extract_single(images_b64, "transcript")
            merged = {}
            if "error" not in transcript_data:
                merged.update(transcript_data)
            if "error" not in diploma_data:
                merged.update(diploma_data)
            if not merged:
                return {"error": "AI processing failed for diploma extraction"}
            return merged

        # Handle category suffixes (e.g., 'prof_development:internship')
        base_category = category.split(':')[0] if ':' in category else category
        user_task = prompts.get(category) or prompts.get(base_category)
        logger.info(f"Processing AI extraction for category: {category} (base: {base_category}). Task found: {user_task is not None}")
        
        if not user_task:
            logger.warning(f"No specific prompt found for category: {category}. Falling back to default.")
            user_task = "Extract data as JSON."

        prompt = f"Category: {category}\n\nTask: {user_task}"
        logger.info(f"Prompt prepared. Length: {len(prompt)}")
        
        max_retries = 2
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

    def classify_document(self, images_b64: list[str], extractor_warnings: list[str]) -> dict:
        """Classifies a document image into one of the predefined categories using Qwen3-VL."""
        if not images_b64:
            return {"category": "unknown", "warnings": extractor_warnings}
            
        try:
            logger.info("Starting document classification...")
            # Use up to 2 pages for classification so the model can detect
            # diploma_with_transcript cases where the grades table starts on page 2
            pages_for_classification = images_b64[:2]
            logger.info(f"Using {len(pages_for_classification)} page(s) for classification")

            response = self.client.chat(
                model=self.model_name,
                messages=[
                    {
                        'role': 'user',
                        'content': self.classification_prompt,
                        'images': pages_for_classification
                    },
                ],
                options={
                    "temperature": 0.0
                }
            )
            
            category = response.message.content.strip().lower()
            # Clean up potential markdown or punctuation
            category = re.sub(r'[^a-z_]', '', category)
            
            valid_categories = [
                "passport", "transcript", "diploma", "motivation", "recommendation",
                "resume", "achievement", "language", "work", "professional_development",
                "certificate", "unknown"
            ]
            
            if category not in valid_categories:
                logger.warning(f"AI returned invalid category: {category}. Defaulting to unknown.")
                return {"category": "unknown", "warnings": extractor_warnings}
                
            logger.info(f"Document classified as: {category}")
            return {"category": category, "warnings": extractor_warnings}
            
        except Exception as e:
            logger.error(f"Error classifying document: {e}", exc_info=True)
            return {"category": "unknown", "warnings": extractor_warnings + [f"Ошибка классификации."]}

    def scan_directory_recursive(self, dir_path: str) -> list[dict]:
        """Recursively scans a directory and classifies all supported files."""
        results = []
        supported_extensions = ('.jpg', '.jpeg', '.png', '.pdf')
        
        if not os.path.isdir(dir_path):
            logger.error(f"Directory not found: {dir_path}")
            return results

        root_abs = os.path.abspath(dir_path)
        
        for root, dirs, files in os.walk(dir_path):
            # Ignore hidden directories
            dirs[:] = [d for d in dirs if not d.startswith('.')]
            
            for file in files:
                if file.startswith('.') or not file.lower().endswith(supported_extensions):
                    continue
                    
                full_path = os.path.abspath(os.path.join(root, file))
                # Calculate relative path from the initial scan directory
                rel_path = os.path.relpath(full_path, root_abs)
                
                logger.info(f"Scanning file: {rel_path}")
                
                try:
                    images_b64, warnings = self.extract_document_features(full_path)
                    res = self.classify_document(images_b64, warnings)
                    
                    results.append({
                        "filePath": full_path,
                        "fileName": rel_path,
                        "category": res.get("category", "unknown"),
                        "warnings": res.get("warnings", []),
                        "error": None
                    })
                except Exception as e:
                    results.append({
                        "filePath": full_path,
                        "fileName": rel_path,
                        "category": "unknown",
                        "warnings": [],
                        "error": str(e)
                    })
                    
        return results

    def _extract_single(self, images_b64: list[str], category: str) -> dict:
        """Runs a single-category extraction (no composite logic). Used internally by process_with_ai."""
        
        prompts = {
            "diploma": (
                "Extract diploma details into a JSON object with exactly these keys: "
                "institution_name, degree_title (must be only 'Bachelor', 'Master', or similar standard degree tier), major, graduation_date (YYYY-MM-DD format), diploma_serial_number. "
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
        }
        user_task = prompts.get(category, "Extract data as JSON.")
        prompt = f"Category: {category}\n\nTask: {user_task}"

        max_retries = 2
        last_error = "Unknown error"
        for attempt in range(max_retries):
            try:
                response = self.client.chat(
                    model=self.model_name,
                    messages=[
                        {'role': 'system', 'content': system_prompt},
                        {'role': 'user', 'content': prompt, 'images': images_b64},
                    ],
                    options={"num_ctx": 32768, "temperature": 0.0}
                )
                data, _ = self._parse_json_response(response.message.content)
                if "error" not in data and data:
                    return self._normalize_and_sanitize(data)
                last_error = data.get("error", "Empty response")
            except Exception as e:
                last_error = str(e)
                logger.error(f"[_extract_single] Error on attempt {attempt+1} for {category}: {e}")
        return {"error": f"Extraction failed for {category}: {last_error}"}

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

    def _get_criterion_guidance(self, code: str) -> str:
        """Returns specific evaluation rubric for known criterion types."""
        code_upper = code.upper()

        if 'MOTIVATION' in code_upper or 'MOT' in code_upper:
            return (
                "Evaluate the motivation letter against ALL of the following criteria:\n"
                "1. Does the applicant explicitly name the target educational PROGRAM?\n"
                "2. Does the applicant explicitly name the target UNIVERSITY?\n"
                "3. Is the letter well-written and grammatically coherent (no obvious errors, clear structure)?\n"
                "4. Does the applicant state a clear GOAL — why this specific program is necessary for them?\n"
                "5. Are specific courses, disciplines, or faculty members mentioned?\n"
                "6. Does the applicant explain why they consider themselves the BEST candidate "
                "(unique skills, experience, achievements)?\n"
                "A letter that fully addresses all six points deserves the maximum score. "
                "Each missing point proportionally lowers the score. "
                "A generic letter that mentions none of these deserves 0."
            )

        if 'RECOMMEND' in code_upper or 'REC' in code_upper:
            return (
                "Evaluate the recommendation letter(s) against these criteria:\n"
                "1. Does the author provide SPECIFIC, CONCRETE examples of the applicant's qualities, "
                "achievements, or situations? (Generic phrases like 'excellent student' or 'highly recommend' "
                "without evidence do NOT qualify.)\n"
                "2. Are the stated reasons for recommending the applicant personalised and detailed?\n"
                "Score high only when the letter contains concrete facts, named situations, or measurable results. "
                "Score low when the letter is vague, formulaic, or indistinguishable from a template."
            )

        return ""

    def score_portfolio(self, criteria: list[dict], applicant_data: dict) -> list[dict]:
        """
        Оценивает категории портфолио абитуриента по заданным критериям.
        Использует только текстовый режим (не требует изображений).
        Возвращает список: [{"code": str, "score": int, "comment": str}]
        """
        # Критерий VIDEO не может быть оценён без просмотра видео — пропускаем
        SKIP_CRITERIA = {'VIDEO'}

        results = []

        for crit in criteria:
            code = crit.get('code', '')
            title = crit.get('title', code)
            max_score = int(crit.get('max_score', 0))

            if code in SKIP_CRITERIA:
                results.append({
                    'code': code,
                    'score': 0,
                    'comment': 'Видео оценивается экспертом вручную'
                })
                continue

            # Формируем текстовое резюме данных абитуриента для этого критерия
            relevant_data = self._build_data_summary(crit, applicant_data)
            guidance = self._get_criterion_guidance(code)
            guidance_block = f"\n\nEvaluation rubric:\n{guidance}" if guidance else ""

            prompt = f"""You are an academic admissions committee member scoring an applicant's portfolio.

Criterion: {title}
Maximum score: {max_score}{guidance_block}

Applicant data relevant to this criterion:
{relevant_data}

Based on the provided data, assign a score from 0 to {max_score}.
If there is no relevant data, assign 0.

Respond with ONLY a JSON object in this exact format:
{{"score": <integer 0 to {max_score}>, "comment": "<brief reasoning in Russian, 1-2 sentences>"}}

Do not add any other text, explanation, or markdown."""

            try:
                response = self.client.chat(
                    model=self.model_name,
                    messages=[{'role': 'user', 'content': prompt}],
                    options={"temperature": 0.0, "num_ctx": 4096}
                )
                content = response.message.content.strip()
                # Извлекаем JSON из ответа
                data, _ = self._parse_json_response(content)
                score = min(max(int(data.get('score', 0)), 0), max_score)
                comment = str(data.get('comment', ''))
                results.append({'code': code, 'score': score, 'comment': comment})
                logger.info(f"[AI Scoring] {code}: {score}/{max_score}")
            except Exception as e:
                logger.error(f"[AI Scoring] Error scoring criterion {code}: {e}")
                results.append({'code': code, 'score': 0, 'comment': f'Ошибка AI-оценки: {e}'})

        return results

    def _build_data_summary(self, criterion: dict, applicant_data: dict) -> str:
        """Формирует текстовое резюме данных абитуриента для конкретного критерия."""
        code = criterion.get('code', '').upper()
        parts = []

        # Маппинг кодов критериев на разделы данных
        if 'EDU' in code or 'DIPLOMA' in code:
            if 'diploma' in applicant_data:
                parts.append(f"Education: {applicant_data['diploma']}")
            if 'transcript' in applicant_data:
                parts.append(f"Transcript/GPA: {applicant_data['transcript']}")
            if 'second_diploma' in applicant_data:
                parts.append(f"Additional diplomas: {applicant_data['second_diploma']}")
            # EDU_ADD covers professional development and certifications too
            if 'ADD' in code:
                if 'prof_development' in applicant_data:
                    parts.append(f"Professional development / training: {applicant_data['prof_development']}")
                if 'certification' in applicant_data:
                    parts.append(f"Certifications: {applicant_data['certification']}")

        if 'ACHIEVE' in code or 'IEEE' in code or 'ADD_ACHIEV' in code:
            if 'achievement' in applicant_data:
                parts.append(f"Achievements: {applicant_data['achievement']}")
            if 'second_diploma' in applicant_data:
                parts.append(f"Additional education: {applicant_data['second_diploma']}")
            # ADD_ACHIEV_COMBINED also covers professional development and certifications
            if 'ADD_ACHIEV' in code:
                if 'prof_development' in applicant_data:
                    parts.append(f"Professional development / training: {applicant_data['prof_development']}")
                if 'certification' in applicant_data:
                    parts.append(f"Certifications: {applicant_data['certification']}")

        if 'MOTIVATION' in code or 'MOT' in code:
            if 'motivation' in applicant_data:
                parts.append(f"Motivation letter: {applicant_data['motivation']}")

        if 'RECOMMEND' in code or 'REC' in code:
            if 'recommendation' in applicant_data:
                parts.append(f"Recommendations: {applicant_data['recommendation']}")

        if 'ENGLISH' in code or 'LANG' in code:
            if 'language' in applicant_data:
                parts.append(f"Language certificate: {applicant_data['language']}")

        if 'WORK' in code or 'PROF' in code:
            if 'work' in applicant_data:
                parts.append(f"Work experience: {applicant_data['work']}")
            if 'prof_development' in applicant_data:
                parts.append(f"Professional development: {applicant_data['prof_development']}")

        # Если специфического маппинга нет — отдаём всё
        if not parts:
            for key, val in applicant_data.items():
                if val:
                    parts.append(f"{key}: {val}")

        if not parts:
            return "No data available for this criterion."

        summary = "\n".join(parts)
        # Ограничиваем длину чтобы не переполнить контекст
        return summary[:3000]

    def generate_annotation(self, applicant_data: dict) -> str:
        """
        Генерирует краткую текстовую аннотацию на абитуриента на основе всех извлечённых данных.
        """
        sections = []

        personal = applicant_data.get("personal_data")
        if personal:
            sections.append(f"Персональные данные: {personal}")

        # Go sends "diploma" (not "education")
        education = applicant_data.get("diploma")
        if education:
            sections.append(f"Базовое образование: {education}")

        transcript = applicant_data.get("transcript")
        if transcript:
            sections.append(f"Академическая успеваемость: {transcript}")

        # Go sends "prof_development", "second_diploma", "certification" separately
        additional_parts = []
        for key in ("prof_development", "second_diploma", "certification"):
            val = applicant_data.get(key)
            if val:
                additional_parts.append(val)
        if additional_parts:
            sections.append(f"Дополнительное образование и опыт работы: {'; '.join(additional_parts)}")

        achievement = applicant_data.get("achievement")
        if achievement:
            sections.append(f"Достижения: {achievement}")

        motivation = applicant_data.get("motivation")
        if motivation:
            sections.append(f"Мотивационное письмо: {motivation}")

        recommendation = applicant_data.get("recommendation")
        if recommendation:
            sections.append(f"Рекомендации: {recommendation}")

        language = applicant_data.get("language")
        if language:
            sections.append(f"Языковой сертификат: {language}")

        data_summary = "\n\n".join(sections) if sections else "Данные отсутствуют."
        data_summary = data_summary[:6000]

        # /no_think disables Qwen3 extended thinking mode so tokens are spent on the annotation itself
        prompt = f"""/no_think
Ты — аналитик приёмной комиссии магистратуры. Составь краткую профессиональную аннотацию на абитуриента на русском языке (3–5 абзацев).

Данные абитуриента:
{data_summary}

Требования к аннотации:
- Структурируй по разделам: персональные данные, образование, опыт/компетенции, мотивация, выводы.
- Используй деловой стиль, без лишних оборотов.
- Если каких-то данных нет, просто пропусти этот раздел.
- Отвечай ТОЛЬКО текстом аннотации, без заголовков, без JSON, без пояснений.

Аннотация:"""

        try:
            response = self.client.chat(
                model=self.model_name,
                messages=[{"role": "user", "content": prompt}],
                options={"temperature": 0.4, "num_predict": 2048, "num_ctx": 8192}
            )
            resp_dict = response.model_dump() if hasattr(response, 'model_dump') else dict(response)
            content = resp_dict.get("message", {}).get("content", "")
            # Strip closed <think>...</think> blocks
            content = re.sub(r'<think>.*?</think>', '', content, flags=re.DOTALL)
            # Strip unclosed <think> block if present (truncated thinking)
            last_think = content.rfind('<think>')
            if last_think != -1 and '</think>' not in content[last_think:]:
                content = content[:last_think]
            return content.strip()
        except Exception as e:
            logger.error(f"[ANNOTATION] Error generating annotation: {e}")
            return f"Ошибка генерации аннотации: {e}"
