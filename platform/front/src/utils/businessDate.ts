import { processDate } from './processDates';

/** All ERP business entry dates use the Beijing calendar day. */
export const businessToday = (): string => processDate(new Date().toISOString());
