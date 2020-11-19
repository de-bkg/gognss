package site

// bern52StaTempl is the template for Bernese 52 STA-files.
const bern52StaTempl = `
STATION INFORMATION FILE FOR BERNESE GNSS SOFTWARE 5.2           {{creationTime}}
--------------------------------------------------------------------------------

FORMAT VERSION: 1.01
TECHNIQUE:      GNSS

TYPE 001: RENAMING OF STATIONS
------------------------------

STATION NAME          FLG          FROM                   TO         OLD STATION NAME      REMARK
****************      ***  YYYY MM DD HH MM SS  YYYY MM DD HH MM SS  ********************  ************************
{{range .}}{{. | encodeTyp1}}
{{end}}
ACOR 13434M001        001  1998 12 06 10 10 00                       ACOR*                 EUREF.SNX
ADAR 19161M001        001  2009 03 04 12 00 00                       ADAR*                 EUREF.SNX
BRON 19686M001        001  2013 01 05 09 00 00  2013 10 25 09 00 00  BRON*                 EUREF.SNX
ZYWI 12220S001        001  2003 01 30 00 00 00                       ZYWI*                 EUREF.SNX


TYPE 002: STATION INFORMATION
-----------------------------

STATION NAME          FLG          FROM                   TO         RECEIVER TYPE         RECEIVER SERIAL NBR   REC #   ANTENNA TYPE          ANTENNA SERIAL NBR    ANT #    NORTH      EAST      UP      DESCRIPTION             REMARK
****************      ***  YYYY MM DD HH MM SS  YYYY MM DD HH MM SS  ********************  ********************  ******  ********************  ********************  ******  ***.****  ***.****  ***.****  **********************  ************************
{{range .}}{{. | encodeTyp2 | html}}{{end}}

ACOR 13434M001        001  1998 12 06 10 10 00  2007 03 17 23 59 00  ASHTECH UZ-12                               999999  ASH700936D_M    SNOW                        999999    0.0000    0.0000    3.0420  A Coruna                EUREF.SNX
ACOR 13434M001        001  2007 03 18 00 00 00  2019 02 04 12 00 00  LEICA GRX1200PRO                            999999  LEIAT504        LEIS                        999999    0.0000    0.0000    3.0460  A Coruna                EUREF.SNX
ACOR 13434M001        001  2019 02 04 12 00 00  2019 10 18 15 00 00  LEICA GR10                                  999999  LEIAT504        LEIS                        999999    0.0000    0.0000    3.0460  A Coruna                EUREF.SNX
ACOR 13434M001        001  2019 10 18 16 00 00                       LEICA GR50                                  999999  LEIAT504        LEIS                        999999    0.0000    0.0000    3.0460  A Coruna                EUREF.SNX
ADAR 19161M001        001  2009 03 04 12 00 00  2018 05 02 10 15 00  LEICA GRX1200+GNSS                          999999  LEIAR25         LEIT                        999999    0.0000    0.0000    0.1888  Aberdaron               EUREF.SNX
ADAR 19161M001        001  2018 05 02 10 35 00                       SEPT POLARX5                                999999  LEIAR25         LEIT                        999999    0.0000    0.0000    0.1888  Aberdaron               EUREF.SNX
AJAC 10077M005        001  2000 01 22 00 00 00  2008 11 26 00 00 00  ASHTECH Z-XII3                              999999  ASH700936A_M    NONE                        999999    0.0000    0.0000    0.0000  Ajaccio                 EUREF.SNX
AJAC 10077M005        001  2008 11 26 00 00 00  2012 12 05 09 00 00  LEICA GRX1200GGPRO                          999999  LEIAT504GG      NONE                        999999    0.0000    0.0000    0.0000  Ajaccio                 EUREF.SNX
AJAC 10077M005        001  2012 12 05 09 00 00  2020 01 21 09 59 00  LEICA GR25                                  999999  TRM57971.00     NONE                        999999    0.0000    0.0000    0.0000  Ajaccio                 EUREF.SNX
AJAC 10077M005        001  2020 01 21 10 00 00                       SEPT POLARX5                                999999  TRM115000.00    NONE                        999999    0.0000    0.0000    0.0000  Ajaccio                 EUREF.SNX
ZYWI 12220S001        001  2003 01 30 00 00 00  2007 11 25 21 00 00  ASHTECH UZ-12                               999999  ASH701945C_M    SNOW                        999999    0.0000    0.0000    0.0000  Zywiec                  EUREF.SNX
ZYWI 12220S001        001  2007 11 27 00 00 00  2014 08 29 10 00 00  TRIMBLE NETR5                               999999  TRM55971.00     TZGD                        999999    0.0000    0.0000    0.0000  Zywiec                  EUREF.SNX
ZYWI 12220S001        001  2014 08 29 11 00 00                       TRIMBLE NETR9                               999999  TRM59900.00     SCIS                        999999    0.0000    0.0000    0.0000  Zywiec                  EUREF.SNX


TYPE 003: HANDLING OF STATION PROBLEMS
--------------------------------------

STATION NAME          FLG          FROM                   TO         REMARK
****************      ***  YYYY MM DD HH MM SS  YYYY MM DD HH MM SS  ************************************************************


TYPE 004: STATION COORDINATES AND VELOCITIES (ADDNEQ)
-----------------------------------------------------
                                            RELATIVE CONSTR. POSITION     RELATIVE CONSTR. VELOCITY
STATION NAME 1        STATION NAME 2        NORTH     EAST      UP        NORTH     EAST      UP
****************      ****************      **.*****  **.*****  **.*****  **.*****  **.*****  **.*****


TYPE 005: HANDLING STATION TYPES
--------------------------------

STATION NAME          FLG  FROM                 TO                   MARKER TYPE           REMARK
****************      ***  YYYY MM DD HH MM SS  YYYY MM DD HH MM SS  ********************  ************************

`
