-- Drop all tables in reverse dependency order
DROP TABLE IF EXISTS public.extracted_fields CASCADE;
DROP TABLE IF EXISTS public.operator_actions CASCADE;
DROP TABLE IF EXISTS public.expert_evaluations CASCADE;
DROP TABLE IF EXISTS public.expert_slots CASCADE;
DROP TABLE IF EXISTS public.evaluation_criteria CASCADE;
DROP TABLE IF EXISTS public.document_processing_queue CASCADE;
DROP TABLE IF EXISTS public.applicants_data_achievements CASCADE;
DROP TABLE IF EXISTS public.applicants_data_education CASCADE;
DROP TABLE IF EXISTS public.applicants_data_identification CASCADE;
DROP TABLE IF EXISTS public.applicants_data_language_training CASCADE;
DROP TABLE IF EXISTS public.applicants_data_motivation CASCADE;
DROP TABLE IF EXISTS public.applicants_data_recommendation CASCADE;
DROP TABLE IF EXISTS public.applicants_data_resume CASCADE;
DROP TABLE IF EXISTS public.applicants_data_transcript CASCADE;
DROP TABLE IF EXISTS public.applicants_data_work_experience CASCADE;
DROP TABLE IF EXISTS public.applicants_data_video CASCADE;
DROP TABLE IF EXISTS public.applicants_document CASCADE;
DROP TABLE IF EXISTS public.applicants CASCADE;
DROP TABLE IF EXISTS public.programs CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;

-- programs
CREATE TABLE public.programs (
    id serial4 NOT NULL,
    title varchar(255) NOT NULL,
    year int4 NOT NULL,
    description text NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    status varchar(20) DEFAULT 'active'::character varying NOT NULL,
    CONSTRAINT programs_pkey PRIMARY KEY (id),
    CONSTRAINT programs_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'completed'::character varying])::text[])))
);

-- users
CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email varchar(255) NOT NULL,
    password varchar(255) NOT NULL,
    role varchar(50) DEFAULT 'user'::character varying NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    last_online timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    first_name varchar(255) NULL,
    last_name varchar(255) NULL,
    phone varchar(20) NULL,
    avatar_path text NULL,
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

-- applicants
CREATE TABLE public.applicants (
    id bigserial NOT NULL,
    program_id int8 NULL,
    status varchar(50) DEFAULT 'uploaded'::character varying NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp NULL,
    aggregated_score numeric(5, 2) DEFAULT 0 NULL,
    scoring_scheme varchar(20) DEFAULT 'auto'::character varying NOT NULL,
    CONSTRAINT applicants_pkey PRIMARY KEY (id),
    CONSTRAINT fk_program FOREIGN KEY (program_id) REFERENCES public.programs(id)
);
CREATE INDEX applicants_index_0 ON public.applicants USING btree (program_id, status);
CREATE INDEX applicants_score_idx ON public.applicants USING btree (aggregated_score DESC);

-- applicants_document
CREATE TABLE public.applicants_document (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    file_type varchar(255) NOT NULL,
    file_name varchar(255) NOT NULL,
    storage_path text NOT NULL,
    uploaded_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status varchar(50) DEFAULT 'uploaded'::character varying NULL,
    processed_at timestamptz NULL,
    CONSTRAINT applicants_document_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_document_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE
);
CREATE INDEX applicants_document_index_0 ON public.applicants_document USING btree (applicant_id);

-- applicants_data_video
CREATE TABLE public.applicants_data_video (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    video_url text NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    CONSTRAINT applicants_data_video_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_video_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE
);

-- document_processing_queue
CREATE TABLE public.document_processing_queue (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    applicant_id int8 NOT NULL,
    document_category varchar(50) NOT NULL,
    file_path varchar(255) NOT NULL,
    priority int4 DEFAULT 5 NOT NULL,
    status varchar(20) DEFAULT 'pending'::character varying NOT NULL,
    error_message text NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    CONSTRAINT document_processing_queue_pkey PRIMARY KEY (id),
    CONSTRAINT document_processing_queue_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE
);
CREATE INDEX idx_document_queue_status_priority ON public.document_processing_queue USING btree (status, priority, created_at);

-- evaluation_criteria
CREATE TABLE public.evaluation_criteria (
    code varchar(50) NOT NULL,
    title varchar(255) NOT NULL,
    max_score int4 NOT NULL,
    type varchar(20) DEFAULT 'BASE'::character varying NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    document_types text[] DEFAULT '{}'::text[] NOT NULL,
    is_mandatory bool DEFAULT false NOT NULL,
    scheme varchar(20) DEFAULT 'default'::character varying NOT NULL,
    program_id int8 NULL,
    CONSTRAINT evaluation_criteria_max_score_check CHECK ((max_score > 0)),
    CONSTRAINT evaluation_criteria_pkey PRIMARY KEY (code),
    CONSTRAINT evaluation_criteria_scheme_check CHECK (((scheme)::text = ANY ((ARRAY['default'::character varying, 'ieee'::character varying])::text[]))),
    CONSTRAINT evaluation_criteria_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.programs(id) ON DELETE CASCADE
);
CREATE INDEX criteria_program_idx ON public.evaluation_criteria USING btree (program_id);

-- expert_evaluations
CREATE TABLE public.expert_evaluations (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    expert_id uuid NOT NULL,
    category varchar(50) NOT NULL,
    score int4 NOT NULL,
    comment text NULL,
    updated_by_id uuid NULL,
    is_admin_override bool DEFAULT false NOT NULL,
    source_info varchar(255) NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status varchar(20) DEFAULT 'DRAFT'::character varying NOT NULL,
    CONSTRAINT expert_evaluations_applicant_id_expert_id_category_key UNIQUE (applicant_id, expert_id, category),
    CONSTRAINT expert_evaluations_pkey PRIMARY KEY (id),
    CONSTRAINT expert_evaluations_score_check CHECK ((score >= 0)),
    CONSTRAINT expert_evaluations_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE
);
CREATE INDEX expert_eval_applicant_idx ON public.expert_evaluations USING btree (applicant_id);

-- expert_slots
CREATE TABLE public.expert_slots (
    slot_number int4 NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    program_id int8 NOT NULL,
    CONSTRAINT expert_slots_pkey PRIMARY KEY (slot_number, program_id),
    CONSTRAINT expert_slots_slot_number_check CHECK (((slot_number >= 1) AND (slot_number <= 3))),
    CONSTRAINT expert_slots_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.programs(id)
);
CREATE UNIQUE INDEX expert_slots_user_program_idx ON public.expert_slots USING btree (user_id, program_id);

-- extracted_fields
CREATE TABLE public.extracted_fields (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    document_id int8 NOT NULL,
    field_name varchar(255) NOT NULL,
    field_value text NULL,
    extracted_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT extracted_fields_pkey PRIMARY KEY (id),
    CONSTRAINT extracted_fields_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT extracted_fields_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- operator_actions
CREATE TABLE public.operator_actions (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    operator_id int8 NOT NULL,
    action_type varchar(100) NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    CONSTRAINT operator_actions_pkey PRIMARY KEY (id),
    CONSTRAINT operator_actions_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE
);

-- applicants_data_achievements
CREATE TABLE public.applicants_data_achievements (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    achievement_title varchar(255) NULL,
    description text NULL,
    date_received date NULL,
    document_path text NULL,
    source varchar(50) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    company varchar(255) NULL,
    end_date date NULL,
    achievement_type varchar(255) NULL,
    CONSTRAINT applicants_data_achievements_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_achievements_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_achievements_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_education
CREATE TABLE public.applicants_data_education (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    institution_name varchar(255) NULL,
    degree_title varchar(255) NULL,
    major varchar(255) NULL,
    graduation_date date NULL,
    diploma_serial_number varchar(255) NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    CONSTRAINT applicants_data_education_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_education_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_education_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_identification
CREATE TABLE public.applicants_data_identification (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    email varchar(255) NULL,
    phone varchar(255) NULL,
    document_number varchar(255) NULL,
    name varchar(255) NULL,
    surname varchar(255) NULL,
    patronymic varchar(255) NULL,
    date_of_birth date NULL,
    gender varchar(50) NULL,
    nationality varchar(255) NULL,
    photo_path text NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    CONSTRAINT applicants_data_identification_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_identification_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_identification_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_language_training
CREATE TABLE public.applicants_data_language_training (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    russian_level varchar(50) NULL,
    english_level varchar(50) NULL,
    certificate_path text NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    exam_name varchar(100) NULL,
    score varchar(50) NULL,
    CONSTRAINT applicants_data_language_training_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_language_training_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_language_training_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_motivation
CREATE TABLE public.applicants_data_motivation (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    reasons_for_applying text NULL,
    experience_summary text NULL,
    career_goals text NULL,
    detected_language varchar(50) NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    main_text text NULL,
    CONSTRAINT applicants_data_motivation_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_motivation_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_motivation_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_recommendation
CREATE TABLE public.applicants_data_recommendation (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    author_name varchar(255) NULL,
    author_position varchar(255) NULL,
    author_institution varchar(255) NULL,
    key_strengths text NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    CONSTRAINT applicants_data_recommendation_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_recommendation_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_recommendation_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_resume
CREATE TABLE public.applicants_data_resume (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    summary text NULL,
    skills text[] NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    CONSTRAINT applicants_data_resume_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_resume_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_resume_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_transcript
CREATE TABLE public.applicants_data_transcript (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    gpa numeric(5, 2) NULL,
    total_credits numeric(10, 2) NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    total_semesters int4 NULL,
    cumulative_gpa numeric(5, 2) NULL,
    cumulative_grade varchar(50) NULL,
    obtained_credits numeric(10, 2) NULL,
    CONSTRAINT applicants_data_transcript_pkey PRIMARY KEY (id),
    CONSTRAINT unique_document_transcript UNIQUE (document_id),
    CONSTRAINT applicants_data_transcript_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_transcript_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- applicants_data_work_experience
CREATE TABLE public.applicants_data_work_experience (
    id bigserial NOT NULL,
    applicant_id int8 NOT NULL,
    country varchar(100) NULL,
    city varchar(100) NULL,
    position varchar(255) NULL,
    company_name varchar(255) NULL,
    start_date date NULL,
    end_date date NULL,
    source varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
    document_id int8 NULL,
    record_type varchar(255) NULL,
    CONSTRAINT applicants_data_work_experience_pkey PRIMARY KEY (id),
    CONSTRAINT applicants_data_work_experience_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.applicants(id) ON DELETE CASCADE,
    CONSTRAINT applicants_data_work_experience_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.applicants_document(id) ON DELETE CASCADE
);

-- Seed: admin user (password = 'password', bcrypt hash)
INSERT INTO public.users (email, password, role, first_name, last_name)
VALUES ('admin@masters.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin', 'Admin', 'User')
ON CONFLICT (email) DO NOTHING;

-- Seed: evaluation criteria (from migration 000023 + 000026)
INSERT INTO public.evaluation_criteria (code, title, max_score, type, document_types, is_mandatory, scheme) VALUES
    ('diploma',          'Диплом о высшем образовании',    20, 'BASE',        ARRAY['diploma'],          true,  'default'),
    ('transcript',       'Транскрипт (средний балл)',       20, 'BASE',        ARRAY['transcript'],       true,  'default'),
    ('work_experience',  'Опыт работы',                    20, 'BASE',        ARRAY['work_experience'],  false, 'default'),
    ('motivation',       'Мотивационное письмо',           15, 'BASE',        ARRAY['motivation'],       true,  'default'),
    ('recommendation',   'Рекомендательное письмо',        10, 'BASE',        ARRAY['recommendation'],   false, 'default'),
    ('achievements',     'Достижения',                     10, 'BASE',        ARRAY['achievements'],     false, 'default'),
    ('language',         'Знание языков',                   5, 'BASE',        ARRAY['language_training'], false, 'default'),
    ('video',            'Видео-презентация',              10, 'ALTERNATIVE', ARRAY['video_presentation'], false, 'default'),
    ('resume',           'Резюме',                         10, 'ALTERNATIVE', ARRAY['resume'],           false, 'default')
ON CONFLICT (code) DO NOTHING;
