'use client';

import { createContext, createElement, useContext, useEffect, useState } from 'react';

export type Lang = 'ru' | 'kk' | 'en';

export const LANGS: { code: Lang; label: string; flag: string }[] = [
  { code: 'ru', label: 'Русский', flag: '🇷🇺' },
  { code: 'kk', label: 'Қазақша', flag: '🇰🇿' },
  { code: 'en', label: 'English', flag: '🇬🇧' },
];

const DICT: Record<Lang, Record<string, string>> = {
  ru: {
    // nav
    nav_home: 'Главная',
    nav_meds: 'Лекарства',
    nav_plans: 'Расписание',
    nav_alerts: 'Оповещения',
    nav_messages: 'Чат',
    nav_patients: 'Пациенты',
    nav_add: 'Добавить',
    logout: 'Выйти',
    // common
    save: 'Сохранить',
    cancel: 'Отмена',
    edit: 'Изменить',
    delete: 'Удалить',
    add: 'Добавить',
    confirm_delete: 'Подтвердите удаление',
    yes: 'Да',
    loading: 'Загружаем...',
    // login
    login_title: 'Вход',
    login_email: 'Email',
    login_password: 'Пароль',
    login_submit: 'Войти',
    login_no_account: 'Нет аккаунта?',
    login_register: 'Зарегистрироваться',
    login_demo: 'Демо-доступ:',
    login_tagline: 'Здоровье под контролем — для вас и ваших близких',
    // register
    register_title: 'Регистрация',
    register_iam: 'Я —',
    role_patient: 'Пациент',
    role_doctor: 'Врач',
    role_family: 'Родственник',
    register_fullname: 'ФИО',
    register_password_hint: 'Пароль (минимум 6 символов)',
    register_phone: 'Телефон',
    register_birth: 'Дата рождения',
    register_submit: 'Создать аккаунт',
    register_creating: 'Создаём...',
    register_have_account: 'Уже есть аккаунт?',
    register_login: 'Войти',
    // home
    greeting_morning: 'Доброе утро',
    greeting_day: 'Добрый день',
    greeting_evening: 'Добрый вечер',
    recent_metrics: 'Последние показатели',
    today_meds: 'Лекарства сегодня',
    today_no_meds: 'На сегодня лекарств нет.',
    quick_entry: 'Быстрый ввод',
    all_metrics: 'Показать все метрики и графики →',
    show_all: 'Все →',
    invite_code_label: 'Код приглашения',
    invite_code_hint:
      'Сообщите этот код врачу или родственнику, чтобы они могли видеть ваши показатели.',
    alerts_new: 'новых оповещений',
    take_dose: 'Принял ✓',
    dose_taken: '✓ Принято',
    // plans
    plans_title: 'Недельное расписание',
    plans_add: 'План',
    plans_edit_title: 'Редактировать',
    plans_new_title: 'Новый план',
    plan_name: 'Название',
    plan_day: 'День',
    plan_time: 'Время',
    plan_type: 'Тип',
    plan_empty: 'На этот день планов нет.',
    plan_confirm: 'Удалить этот план?',
    plan_type_doctor_visit: 'Визит к врачу',
    plan_type_take_med: 'Приём лекарства',
    plan_type_rest: 'Отдых',
    plan_type_other: 'Другое',
    // meds
    meds_title: 'Мои лекарства',
    meds_add_title: 'Название',
    meds_dosage: 'Дозировка',
    meds_times: 'Время приёма (через запятую)',
    meds_empty: 'Пока нет активных лекарств.',
    meds_confirm: 'Удалить это лекарство?',
    // bmi
    bmi_label: 'BMI',
    bmi_set_height: 'Укажите свой рост',
    bmi_set_btn: 'Указать',
    bmi_need_weight: 'добавьте замер веса',
    bmi_under: 'Недостаточный вес',
    bmi_normal: 'Норма',
    bmi_over: 'Избыточный вес',
    bmi_obese: 'Ожирение',
    // profile
    profile_title: 'Профиль',
    profile_devices: 'IoT-устройства',
    device_watch: 'Умные часы',
    device_bp: 'Тонометр',
    device_thermo: 'Термометр',
    device_connected: 'Подключено',
    device_disconnected: 'Отключено',
    device_connect: 'Подключить',
    device_disconnect: 'Отключить',
    profile_invite: 'Код-приглашение',
    profile_language: 'Язык',
    // onboarding
    onboard_title: 'О вас',
    onboard_sub: 'Заполните, чтобы мы могли лучше считать нормы',
    onboard_height: 'Рост (см)',
    onboard_weight: 'Вес (кг)',
    onboard_chronic: 'Хронические заболевания',
    onboard_bp_norm: 'Норма давления',
    onboard_meds: 'Назначенные лекарства',
    onboard_submit: 'Сохранить и продолжить',
    onboard_saving: 'Сохраняем...',
  },
  kk: {
    nav_home: 'Басты',
    nav_meds: 'Дәрілер',
    nav_plans: 'Кесте',
    nav_alerts: 'Хабарлау',
    nav_messages: 'Чат',
    nav_patients: 'Пациенттер',
    nav_add: 'Қосу',
    logout: 'Шығу',
    save: 'Сақтау',
    cancel: 'Болдырмау',
    edit: 'Өзгерту',
    delete: 'Өшіру',
    add: 'Қосу',
    confirm_delete: 'Өшіруді растаңыз',
    yes: 'Иә',
    loading: 'Жүктелуде...',
    login_title: 'Кіру',
    login_email: 'Email',
    login_password: 'Құпиясөз',
    login_submit: 'Кіру',
    login_no_account: 'Аккаунт жоқ па?',
    login_register: 'Тіркелу',
    login_demo: 'Демо-қолжетімділік:',
    login_tagline: 'Сіздің және жақындарыңыздың денсаулығы бақылауда',
    register_title: 'Тіркелу',
    register_iam: 'Мен —',
    role_patient: 'Пациент',
    role_doctor: 'Дәрігер',
    role_family: 'Туысқан',
    register_fullname: 'Аты-жөні',
    register_password_hint: 'Құпиясөз (кемінде 6 таңба)',
    register_phone: 'Телефон',
    register_birth: 'Туған күні',
    register_submit: 'Аккаунт жасау',
    register_creating: 'Жасалуда...',
    register_have_account: 'Аккаунт бар ма?',
    register_login: 'Кіру',
    greeting_morning: 'Қайырлы таң',
    greeting_day: 'Қайырлы күн',
    greeting_evening: 'Қайырлы кеш',
    recent_metrics: 'Соңғы көрсеткіштер',
    today_meds: 'Бүгінгі дәрілер',
    today_no_meds: 'Бүгінге дәрілер жоқ.',
    quick_entry: 'Жылдам енгізу',
    all_metrics: 'Барлық көрсеткіштер мен графиктер →',
    show_all: 'Барлығы →',
    invite_code_label: 'Шақыру коды',
    invite_code_hint:
      'Бұл кодты дәрігеріңізге немесе туысқаныңызға беріңіз, олар көрсеткіштеріңізді көре алу үшін.',
    alerts_new: 'жаңа хабарлау',
    take_dose: 'Қабылдадым ✓',
    dose_taken: '✓ Қабылданды',
    plans_title: 'Апталық кесте',
    plans_add: 'Жоспар',
    plans_edit_title: 'Өзгерту',
    plans_new_title: 'Жаңа жоспар',
    plan_name: 'Атауы',
    plan_day: 'Күн',
    plan_time: 'Уақыт',
    plan_type: 'Түрі',
    plan_empty: 'Бұл күнге жоспар жоқ.',
    plan_confirm: 'Бұл жоспарды өшіру керек пе?',
    plan_type_doctor_visit: 'Дәрігерге бару',
    plan_type_take_med: 'Дәрі ішу',
    plan_type_rest: 'Демалыс',
    plan_type_other: 'Басқа',
    meds_title: 'Менің дәрілерім',
    meds_add_title: 'Атауы',
    meds_dosage: 'Дозасы',
    meds_times: 'Қабылдау уақыты (үтір арқылы)',
    meds_empty: 'Әзірге белсенді дәрі жоқ.',
    meds_confirm: 'Бұл дәріні өшіру керек пе?',
    bmi_label: 'ДСИ',
    bmi_set_height: 'Бойыңызды көрсетіңіз',
    bmi_set_btn: 'Қосу',
    bmi_need_weight: 'салмақ өлшемін қосыңыз',
    bmi_under: 'Жетіспейтін салмақ',
    bmi_normal: 'Қалыпты',
    bmi_over: 'Артық салмақ',
    bmi_obese: 'Семіздік',
    profile_title: 'Профиль',
    profile_devices: 'IoT-құрылғылар',
    device_watch: 'Ақылды сағат',
    device_bp: 'Тонометр',
    device_thermo: 'Термометр',
    device_connected: 'Қосылған',
    device_disconnected: 'Ажыратылған',
    device_connect: 'Қосу',
    device_disconnect: 'Ажырату',
    profile_invite: 'Шақыру коды',
    profile_language: 'Тіл',
    onboard_title: 'Өзіңіз туралы',
    onboard_sub: 'Нормаларды дұрыс есептеу үшін толтырыңыз',
    onboard_height: 'Бойы (см)',
    onboard_weight: 'Салмағы (кг)',
    onboard_chronic: 'Созылмалы аурулар',
    onboard_bp_norm: 'Қалыпты қысым',
    onboard_meds: 'Тағайындалған дәрілер',
    onboard_submit: 'Сақтап, жалғастыру',
    onboard_saving: 'Сақталуда...',
  },
  en: {
    nav_home: 'Home',
    nav_meds: 'Meds',
    nav_plans: 'Schedule',
    nav_alerts: 'Alerts',
    nav_messages: 'Chat',
    nav_patients: 'Patients',
    nav_add: 'Add',
    logout: 'Logout',
    save: 'Save',
    cancel: 'Cancel',
    edit: 'Edit',
    delete: 'Delete',
    add: 'Add',
    confirm_delete: 'Confirm deletion',
    yes: 'Yes',
    loading: 'Loading...',
    login_title: 'Login',
    login_email: 'Email',
    login_password: 'Password',
    login_submit: 'Sign in',
    login_no_account: "Don't have an account?",
    login_register: 'Register',
    login_demo: 'Demo access:',
    login_tagline: 'Health under control — for you and your loved ones',
    register_title: 'Register',
    register_iam: 'I am —',
    role_patient: 'Patient',
    role_doctor: 'Doctor',
    role_family: 'Family',
    register_fullname: 'Full name',
    register_password_hint: 'Password (at least 6 chars)',
    register_phone: 'Phone',
    register_birth: 'Date of birth',
    register_submit: 'Create account',
    register_creating: 'Creating...',
    register_have_account: 'Already have an account?',
    register_login: 'Sign in',
    greeting_morning: 'Good morning',
    greeting_day: 'Good afternoon',
    greeting_evening: 'Good evening',
    recent_metrics: 'Latest readings',
    today_meds: "Today's medications",
    today_no_meds: 'No medications for today.',
    quick_entry: 'Quick entry',
    all_metrics: 'Show all metrics and charts →',
    show_all: 'All →',
    invite_code_label: 'Invite code',
    invite_code_hint:
      'Share this code with your doctor or family so they can see your readings.',
    alerts_new: 'new alerts',
    take_dose: 'Taken ✓',
    dose_taken: '✓ Taken',
    plans_title: 'Weekly schedule',
    plans_add: 'Plan',
    plans_edit_title: 'Edit',
    plans_new_title: 'New plan',
    plan_name: 'Title',
    plan_day: 'Day',
    plan_time: 'Time',
    plan_type: 'Type',
    plan_empty: 'No plans for this day.',
    plan_confirm: 'Delete this plan?',
    plan_type_doctor_visit: 'Doctor visit',
    plan_type_take_med: 'Take medicine',
    plan_type_rest: 'Rest',
    plan_type_other: 'Other',
    meds_title: 'My medications',
    meds_add_title: 'Name',
    meds_dosage: 'Dosage',
    meds_times: 'Times of day (comma-separated)',
    meds_empty: 'No active medications yet.',
    meds_confirm: 'Delete this medication?',
    bmi_label: 'BMI',
    bmi_set_height: 'Enter your height',
    bmi_set_btn: 'Set',
    bmi_need_weight: 'add a weight reading',
    bmi_under: 'Underweight',
    bmi_normal: 'Normal',
    bmi_over: 'Overweight',
    bmi_obese: 'Obese',
    profile_title: 'Profile',
    profile_devices: 'IoT devices',
    device_watch: 'Smart watch',
    device_bp: 'BP monitor',
    device_thermo: 'Thermometer',
    device_connected: 'Connected',
    device_disconnected: 'Disconnected',
    device_connect: 'Connect',
    device_disconnect: 'Disconnect',
    profile_invite: 'Invite code',
    profile_language: 'Language',
    onboard_title: 'About you',
    onboard_sub: 'Fill in so we can compute your personal baseline better',
    onboard_height: 'Height (cm)',
    onboard_weight: 'Weight (kg)',
    onboard_chronic: 'Chronic conditions',
    onboard_bp_norm: 'Normal blood pressure',
    onboard_meds: 'Prescribed medications',
    onboard_submit: 'Save & continue',
    onboard_saving: 'Saving...',
  },
};

const STORAGE_KEY = 'lang';

function detectInitialLang(): Lang {
  if (typeof window === 'undefined') return 'ru';
  const stored = localStorage.getItem(STORAGE_KEY) as Lang | null;
  if (stored && DICT[stored]) return stored;
  const nav = navigator.language.slice(0, 2).toLowerCase();
  if (nav === 'kk') return 'kk';
  if (nav === 'en') return 'en';
  return 'ru';
}

interface I18nCtx {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: string) => string;
}

const Ctx = createContext<I18nCtx | null>(null);

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>('ru');

  useEffect(() => {
    setLangState(detectInitialLang());
  }, []);

  function setLang(l: Lang) {
    setLangState(l);
    if (typeof window !== 'undefined') localStorage.setItem(STORAGE_KEY, l);
  }

  function t(key: string): string {
    return DICT[lang][key] ?? DICT.ru[key] ?? key;
  }

  return createElement(Ctx.Provider, { value: { lang, setLang, t } }, children);
}

export function useI18n(): I18nCtx {
  const ctx = useContext(Ctx);
  if (!ctx) {
    // Fallback (e.g. during hydration): return a no-op that uses ru dict.
    return {
      lang: 'ru',
      setLang: () => {},
      t: (k: string) => DICT.ru[k] ?? k,
    };
  }
  return ctx;
}
