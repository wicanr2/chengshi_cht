/* 1993 年的原始碼碰上現代 glibc 的相容層。
 * 只補「當年的標頭有、現在沒有」的東西，不改任何語意。
 * 注入方式是 gcc wrapper 的 -include，封存本身一個 byte 都不動。 */
#ifndef SIMCITY_ORACLE_COMPAT_H
#define SIMCITY_ORACLE_COMPAT_H

/* SVID matherr 的例外代碼。glibc 2.27 之後從 <math.h> 移除，
 * 但 tclx 的 tclxfmat.c 仍在 switch 裡用它們。數值取自 SVID3 的定義。 */
#ifndef DOMAIN
#define DOMAIN    1
#endif
#ifndef SING
#define SING      2
#endif
#ifndef OVERFLOW
#define OVERFLOW  3
#endif
#ifndef UNDERFLOW
#define UNDERFLOW 4
#endif
#ifndef TLOSS
#define TLOSS     5
#endif
#ifndef PLOSS
#define PLOSS     6
#endif


/* SVID 的 matherr 回呼參數型別，同樣被 glibc 移除。
 * 欄位順序取自 SVID3；tclx 的 tclxmerr.c 只讀 name 與 type。 */
#ifndef SIMCITY_ORACLE_STRUCT_EXCEPTION
#define SIMCITY_ORACLE_STRUCT_EXCEPTION
struct exception {
    int    type;
    char  *name;
    double arg1;
    double arg2;
    double retval;
};
#endif

#endif
