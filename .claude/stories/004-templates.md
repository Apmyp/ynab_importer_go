---
id: "004"
title: "Specify more templates"
branch: "feature/004-templates"
---

# Specify more templates

## Description

После выполнения команды `missing_templates` я выбрал несколько из них и перечислил ниже где расписал как их правильно читать

Некоторые сообщения не будут иметь шаблона никогда, их нужно пометить как сообщения без шаблоны и не выводить в следующий раз при команде missing_templates. Таким образом мы вместе будем обрабатывать сообщения и всем достанется решение: шаблон или без шаблона. Сообщения приведенные ниже не полное совпадние, а могут быть включены в полное. Я копирую только часть

Сообщения могут быть слегка динамическими (содержать дату, пароль, комментарий, номер карты). Определи похожие сообщения чтобы правильно выявить уникальные шаблоны.

Сообщение: Vas privetstvuet servis opoveshenia ot MAIB
Результат: игнор

Сообщение: Oper.: Ostatok
Результат: игнор

Сообщение: Autentificarea Dvs. in sistemul Eximbank Online a fost inregistrata la
Результат: игнор

Сообщение: Parola de unica folosinta pentru tranzactia cu ID-ul
Результат: игнор

Сообщение: OTP-ul pentru Plati din Exim Personal este
Результат: игнор

Сообщение: Va multumim ca ati ales serviciul Eximbank SMS Info.
Результат: игнор

Сообщение: Parola de Unica Folosinta (OTP) a Dvs. pentru logare este
Результат: игнор

Сообщение: Parola:219281 Card 9..7890
Результат: игнор

Сообщение: Parola:866952 Card 9..7890
Результат: игнор - parola:* может быть любой

Сообщение: Tranzactie esuata,
Результат: игнор

Сообщение: Tranzactia din  din contul * in contul * in suma de * a fost Executata
Результат: игнор

Сообщение: Anulare tranzactie
Результат: игнор

Сообщение: Acesta este momentul pe care il asteptai!
Результат: игнор

Сообщение: Parola Dvs. este aP1qaBkI
Результат: игнор

Сообщение: Vrei un card pentru copilul tau? 
Результат: игнор

Сообщение: Refinanteaza creditele de consum de la alte
Результат: игнор

Сообщение: Profita acum! Credit PERSONAL sau MAGNIFIC
Результат: игнор

Сообщение: In data de 29.11 la 10:00-12:00 vor fi lucrari de mentenanta la Internet Banking si Mobile Banking
Результат: игнор

Сообщение: Cardul Eximbank ****7890 a fost adaugat cu success in Apple Pay.l
Результат: игнор

Сообщение: [2025-06-16 18:23:40] Me: A
Результат: игнор

Сообщение: Suplinire cont Card 9..7890, Data 01.09.2025 15:17:03, Suma 2610 MDL, Detalii Rambursarea sumelor in baza ordinelor nr, 34A-36A din 29., Disponibil 15800.60 MDL
Результат: пополнение моего счета - Suplinire cont Card {CARD_NUMBER}, Data {DATE}, Suma {AMOUNT} {CURRENCY}, {COMMENT}, Disponibil 15800.60 MDL

Сообщение: Debitare cont Card 9..7890, Data 19.06.2024 16:41:08, Suma 876.6 MDL, Detalii Plata OP-OP8888777766665555/ INTERN : PENTRU MPAY, Contrac, Disponibil 7100.40 MDL
Результат: шаблон, списание с моей карты. Debitare cont Card {CARD_NUMBER}, Data {DATE}, Suma {AMOUNT} {CURRENCY}, {COMMENT}, Disponibil 7100.40 MDL

Сообщение: Debitare cont Card 9..7890, Data 08.04.2024 09:27:01, Suma 9.65 MDL, Detalii Comision serviciu SMS pentru cardul nr. 199458, Disponibil 38400.60 MDL
Результат: нужен шаблон. debitare = списание с моей карты 7890 в дату 08.04.2024 09:27:01, сумма 9.65, валюта MDL. Комментарий Detalii Comision serviciu SMS pentru cardul nr. 199458, остаток на счете 38400.60

Сообщение: Tranzactie reusita, Data 13.04.2024 13:20:30, Card 9..7890, Suma 91.91 MDL, Locatie MAIB GROCERY STORE BETA>CHISINAU, MDA, Disponibil 31200.80 MDL
Результат: нужен шаблон, пример бери из debitare. Только Tranzactie reusita тоже означает списание с моего счета

Сообщение: Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 93719.33 MDL, Detalii Plata salariala luna aprilie, Disponibil 88700.25 MDL
Результат: тоже что и предыдущее, только suplinire означает пополнение моего счета

Сообщение: Suplinire cont Card 9..7890, Data 13.01.2025 16:13:56, Suma 990 RUB, Detalii ONLINE SERVICE GAMMA> 44712345678, GBR
Результат: тоже что и предыдущее, только suplinire означает пополнение моего счета - Suplinire cont Card {CARD_NUMBER}, Data {DATE}, Suma {AMOUNT} {CURRENCY}, Detalii {COMMENT}, Disponibil 88700.25 MDL

Сообщение: Suplinire cont Card 9..7890, Data 29.04.2024 16:18:01, Suma 93719.33 MDL, Detalii Plata salariala luna aprilie, Disponibil 88700.25 MDL
Результат: тоже что и предыдущее, только suplinire означает пополнение моего счета

Сообщение: Debitare cont Card 9..7890, Data 19.06.2024 16:41:08, Suma 876.6 MDL, Detalii Plata OP-OP8888777766665555/ INTERN : PENTRU MPAY, Contrac, Disponibil 7100.40 MDL
Результат: списание с моего счета - Debitare cont Card {CARD_NUMBER}, Data {DATE}, Suma {AMOUNT} {CURRENCY}, Detalii {COMMENT}, Disponibil 7100.40 MDL

## Acceptance Criteria

- [ ] Все сообщения и действия учтены, ничего непропущено
- [ ] Команда missing_templates выводит только неразрешенные шаблоны чтобы сократить список

## Definition of Done

- [ ] All tests pass
- [ ] Test coverage > 90%
- [ ] Code compiles without errors
- [ ] Code is formatted with gofmt
- [ ] Following go best practices 
