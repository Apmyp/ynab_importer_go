---
id: "009"
title: "Support system installation"
branch: "feature/009-system-installation"
---

# YNAB

## Description

`ynab_importer_go system_install` команда должна установить скрипт в систему таким образом чтобы он вызывал сам себя каждый час

## Acceptance Criteria

- [ ] Если ОС не МакОС, то выход
- [ ] Если МакОС, то идем дальше
- [ ] Проверяем если можно использовать launchtl - дальше если может - выход если не можем
- [ ] Добавляем эту программу в launchtl от лица текущего пользователя чтобы каждый час делался синк
- [ ] Указываем пути для логов чтобы делать дебаг если что-то со скриптом пошло не так
- [ ] Команда `system_uninstall` безопасно уберет команду из крона на каждый час
- [ ] Опционально: сделай баш скрипт который бы запускал синк если так будет легче подключить его к launchtl

## Definition of Done

- [ ] All tests pass
- [ ] Test coverage > 90%
- [ ] Code compiles without errors
- [ ] Code is formatted with gofmt
- [ ] Following go best practices 
