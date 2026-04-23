system_prompt = (
    """You are a data extraction assistant. Follow these rules:
    1. Always output exactly one JSON object and nothing else.
    2. Use only the specified field names. Do not change their order or casing.
    3. Do not include any explanations, text, or markdown—only the raw JSON (no code fences).
    4. If a value is missing, use null or an empty string "". If you are not sure and must guess, prefix the value with '~'.
    5. Ensure the JSON is valid (keys in double quotes, etc.).
    6. DO NOT use <think> tags or output reasoning steps. Output the JSON block IMMEDIATELY."""
)

extraction_prompts = {
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

classification_prompt = """
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
        (5) any other document confirming English language proficiency submitted for expert review.
        (6) official letter or certificate from a university stating that the language of instruction 
            was English (e.g., "Medium of Instruction is English", "обучение велось на английском языке",
            "language of instruction: English", "taught entirely in English").
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
    12. University letter/certificate stating that the program was taught in English 
        (medium of instruction = English) → language, NOT diploma, NOT transcript, NOT certificate

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
    Document says "This is to certify that the student completed the Bachelor's program taught entirely in English, University of XYZ" → language
    Document says "Medium of Instruction Certificate: all courses were conducted in English" → language
    Document says "Настоящим подтверждается, что обучение в университете велось на английском языке" → language
    Document says "Language of Instruction: English, Faculty of Computer Science, University of Warsaw" → language

    Category:
"""