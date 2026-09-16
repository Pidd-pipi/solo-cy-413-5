import type {Plan} from '../types'; let current:Plan|undefined; export const planStore={get:()=>current,set:(p?:Plan)=>{current=p},clear:()=>{current=undefined}};
