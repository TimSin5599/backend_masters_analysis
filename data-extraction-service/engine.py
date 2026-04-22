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
        self.client = ollama.Client(host=os.getenv("OLLAMA_HOST", "http://host.docker.internal:11434"), timeout=300)
        self.model_name = "qwen3-vl:8b"
        self.classification_prompt = """
            You are a document classification system for university applicants. Analyze the document image and determine its category.

            AVAILABLE CATEGORIES:
            - passport — national passport (contains "паспорт", series/number, personal data, photo page)
            - transcript — standalone academic transcript with NO degree/diploma award statement. Contains only a grades/marks table ("ведомость", "зачётная книжка", "Official Transcript", list of subjects with scores). Choose this ONLY when there is a grades table but NO diploma header or degree award text.
            - diploma — any document that awards or confirms a completed degree/education. Includes:
                (1) standalone diploma with no grades table
                (2) combined documents: "ПРИЛОЖЕНИЕ К ДИПЛОМУ", "выписка из диплома", "Diploma Supplement" (DS), Indian semester marksheets attached to degree, Chinese 成绩单 combined with degree certificate, any degree certificate followed by course/grade table
                Essentially: if a document contains a degree award statement, classify it as diploma regardless of whether it also has a grades table.
            - motivation — motivation letter or personal statement (free-form text explaining why applicant chose this program)
            - recommendation — recommendation or reference letter (written by a third person about the applicant, contains "рекомендую", "рекомендательное письмо")
            - resume — resume or CV (contains work experience, skills, "опыт работы", contact info)
            - achievement — proof of competition/olympiad wins ("диплом победителя/призёра"), scientific works, patents ("патент"), grants ("грант"), author certificates ("авторское свидетельство"), implementation acts ("акт о внедрении"), project activity documents with description of role/scale/domain
            - language — document confirming English language proficiency. Includes:
                (1) standardized test certificates: IELTS, TOEFL, Cambridge (FCE, CAE, CPE), TOEIC, and similar from list 3.5;
                (2) HSE internal English exam results;
                (3) diploma transcript (выписка из диплома) showing English language discipline with credits and grade;
                (4) certificate of completion of English language courses or programs;
                (5) any other document confirming English language proficiency submitted for expert review.
                NOT for second/other foreign languages (French, German, Chinese, etc.)
            - work — work experience document (employment record "трудовая книжка", work contract, reference from employer about job position and responsibilities)
            - professional_development — professional certification, participation in schools/conferences ("конференция", "школа"), second foreign language certificate (NOT English), advanced training ("повышение квалификации"), professional courses
            - certificate — completion certificate for computer, IT, or online courses (Udemy, Coursera, edX, Stepik, Skillbox, Google, Microsoft, etc.), any document titled "Certificate of Completion" or "сертификат" about a specific course or skill that does NOT fall into language or professional_development categories
            - unknown — unreadable, blank, or does not match any category above

            DISAMBIGUATION RULES:
            1. Competition diploma ("диплом победителя", "диплом призёра", "диплом лауреата") → achievement, NOT diploma
            2. Any document with a degree award statement → diploma (even if it also contains a grades table)
            2a. "ПРИЛОЖЕНИЕ К ДИПЛОМУ", "Diploma Supplement", "выписка из диплома" → diploma
            2b. Grades table WITH no degree award statement → transcript
            3. IELTS / TOEFL / Cambridge / any English proficiency document → language, NOT certificate and NOT professional_development
            4. Second foreign language (French, German, Chinese, etc.) → professional_development, NOT language
            5. Document titled "сертификат" or "Certificate of Completion" about a professional skill, tool, or course → certificate
            6. Scientific article, thesis, research paper → achievement
            7. Patent, grant, author certificate → achievement
            8. Online course completion certificate (Udemy, Coursera, edX, Stepik, Google, Microsoft, etc.) → certificate, NOT professional_development
            9. Document confirming completion of English language courses or programs → language, NOT certificate
            10. Diploma transcript (выписка из диплома) that includes English language subject with grade and credits → language, NOT transcript
            11. Any document explicitly confirming English language proficiency (even if not a standardized test) → language

            STRICT RULES:
            - Reply with ONLY the category name from the list above, nothing else
            - Do NOT add explanations, punctuation, or extra words
            - If unsure between two categories, apply DISAMBIGUATION RULES first, then choose the most likely one
            - If the image is unreadable or empty → unknown

            EXAMPLES:
            Document says "ДИПЛОМ победителя олимпиады по математике" → achievement
            Document says "ПРИЛОЖЕНИЕ К ДИПЛОМУ о высшем образовании" with subjects table → diploma
            Document says "Diploma Supplement" (European format) with ECTS credits and grades table → diploma
            Document says "выписка из диплома" with list of disciplines and scores → diploma
            Document says "ДИПЛОМ бакалавра" on page 1, page 2 shows table with дисциплина/оценка columns → diploma
            Document from Indian university shows "Bachelor of Technology" degree info + semester-wise marks table → diploma
            Document from Chinese university shows degree certificate + 成绩单 (grade table) → diploma
            Document shows "Bachelor of Science, University of XYZ, GPA 3.8" followed by course list with letter grades → diploma
            Document shows only diploma header with university name, qualification, graduation year, serial number — no table → diploma
            Document says "ДИПЛОМ бакалавра, Московский государственный университет" with no grades table → diploma
            Document shows only a table of subjects and grades with no degree award statement → transcript
            Document says "Certificate of Completion, English Language Course, intermediate level" → language
            Document shows diploma transcript with subject "Английский язык", grade "отлично", 4 credits → language
            Document says "IELTS Academic, Overall Band Score 7.5" → language
            Document says "HSE English Exam Results, score 78/100" → language
            Document says "Сертификат об окончании курсов английского языка, уровень B2" → language
            Document says "Certificate of Completion, French Language Course" → professional_development
            Document says "СЕРТИФИКАТ, настоящим подтверждается участие в конференции" → professional_development
            Document says "Certificate of Completion, Bootstrap & jQuery Certification Course for Beginners, Udemy" → certificate
            Document says "Certificate of Completion, Master Course in Cloud Computing and Cloud Architecture 2.0, Udemy" → certificate
            Document says "Course Certificate, HTML, CSS, and Javascript for Web Developers, Johns Hopkins University, Coursera" → certificate
            Document says "Course Certificate, Technical Support Fundamentals, Google, Coursera" → certificate
            Document says "Certificate of Completion, CSS And Javascript Crash Course, Udemy" → certificate
            Document shows table with subjects and grades → transcript
            Document says "Патент на изобретение № 2023..." → achievement

            Category:
        """
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
        prompts = {
            "passport": (
                "Extract passport data into a JSON object. IMPORTANT: Use exactly these keys: "
                "\"patronymic\" (look for 'Father Name' or 'Patronymic' field only), "
                "\"document_number\" (serial number or passport number), \"date_of_birth\" (YYYY-MM-DD), \"nationality\", \"gender\" (M or F). "
                "Do NOT extract name or surname fields. "
                "If 'Father Name' or 'Patronymic' is present, map it to 'patronymic'. "
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
                "Fields to extract:\n"
                "- achievement_type: always 'certificate' unless clearly stated otherwise\n"
                "- achievement_title: the exact course or certificate name as written\n"
                "- company: the authorizing organization (e.g. Google, Johns Hopkins University), NOT the platform (not Udemy, not Coursera)\n"
                "  If no authorizing organization is mentioned, use the platform name (Udemy, Coursera, etc.)\n"
                "- date_received: completion date in YYYY-MM-DD format. If only month and year are given, use the 1st as the day.\n\n"
                "Return strictly this JSON format:\n"
                "{\"achievement_type\": \"...\", \"achievement_title\": \"...\", \"company\": \"...\", \"date_received\": \"YYYY-MM-DD\"}\n"
                "Do not add explanation, markdown, or any text outside the JSON."
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
        system_prompt = (
            """You are a data extraction assistant. Follow these rules:
            1. Always output exactly one JSON object and nothing else.
            2. Use only the specified field names. Do not change their order or casing.
            3. Do not include any explanations, text, or markdown—only the raw JSON (no code fences).
            4. If a value is missing, use null or an empty string "". If you are not sure and must guess, prefix the value with '~'.
            5. Ensure the JSON is valid (keys in double quotes, etc.).
            6. DO NOT use <think> tags or output reasoning steps. Output the JSON block IMMEDIATELY."""
        )
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

            prompt = f"""You are an academic admissions committee member scoring an applicant's portfolio.

Criterion: {title}
Maximum score: {max_score}

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
                parts.append(f"Additional education: {applicant_data['second_diploma']}")

        if 'ACHIEVE' in code or 'IEEE' in code or 'ADD_ACHIEV' in code:
            if 'achievement' in applicant_data:
                parts.append(f"Achievements: {applicant_data['achievement']}")
            if 'second_diploma' in applicant_data:
                parts.append(f"Additional education: {applicant_data['second_diploma']}")

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
