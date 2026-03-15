package telegram

const (
	welcomeMsg   = "Добро пожаловать в универсального бота для квизов! Пришлите мне JSON квиза или используйте ссылку чтобы присоединиться к квизу.\n\nИспользуйте:\n/myquizzes для просмотра истории загруженных квизов,\n/myquizruns для просмотра всех запусков квизов"
	underfindMsg = "К сожалению, я не могу расспознать такую команду. Используйте /start для начала работы, /myquizzes для просмотра истории загруженных квизов или /myquizruns для просмотра всех запусков квизов."
	logicalError = "При работе программы возникла логическая ошибка. Обратитесь к разработчику или попробуйте снова позже."

	emptyRunID         = "Вы вызвали \"/start join\", но не передали ID. Вызовите \"/start join_RUNID\", где RUNID - корректный ID квиза."
	underfindID        = "Вы вызвали \"/start join\", но передали несуществующий ID. Вызовите \"/start join_RUNID\", где RUNID - корректный ID квиза."
	underfindIDsession = "ID квиза в вашей сессии несуществует. Вероятно оно устарело. Вызовите \"/start join_RUNID\", где RUNID - корректный ID квиза."
	alreadyJoined      = "Вы уже присоедеинились к квизу с ID: %s."
	regDataReq         = "Для регистрации необходимо ввести %q."
	successfulReg      = "Регистрация на квиз %q с ID: %s, прошла успешно! Ожидайте начала квиза."
	errorJoined        = "К сожалению, при попытке подключения к квизу с ID %q произошла ошибка: %v. Попробйте снова позже."

	answerWithoutConnectToRun = "Вы ещё не подключены ни к одну из квизов. Чтобы отвечать на вопросы, подключитесь к квизу по ссылке."
	unsuccessfulSubmit        = "К сожалению, при регистрации ответа возникла ошибка: %v."
	successfulSubmit          = "✅ Ответ принят!"
	questionExplanation       = "Пояснения:\n%s"

	invalidFormatOfFile = "Пожалуйста, отправьте файл фортмата .json."
	errorFile           = "Возникла ошибка при обработке файла: %v."
	errorStructFile     = "В структуре файла ошибка. Убедитесь в корректности содержимого файла."

	errorStartLobby    = "При попытке запуска лобби возникла ошибка: %v."
	linkPattern        = "https://t.me/%s?start=join_%s"
	successfulFileLoad = "Квиз %q загружен!\nID запуска: %s\n\nСсылка для участников:\n%s\n\nКоличество участников: %d\n"

	unsuccessfulRunQuiz = "При попытке запуска квиза возникла ошибка: %v."
	successfulRunQuiz   = "Квиз запущен!"
	successfulStopQuiz  = "Квиз остановлен!"

	cancelParticipantMsg = "По инициативе автора, квиз %q был остановлен."
	cancelLobby          = "Квиз %q, с ID: %s, остановлен. Лобби закрыто."
	cancelRunning        = "Квиз %q, с ID: %s, остановлен во время проведения. Участники уведомлены."

	letters           = "ABCDEF"
	endingA           = "а\n\n"
	endingB           = "ы\n\n"
	endingC           = "ов\n"
	endingD           = "а\n"
	penultimateOne    = "или "
	questionNum       = "Вопрос №%d\n\n"
	answerLetter      = "\n%s. "
	timePerQuestion   = "⏱ %d секунд"
	actionsIndication = "Выберите букву, соответствующую верному ответу, по вашему мнению: "

	quizResultsTitle  = "🏆 Результаты квиза: %s\n\n"
	personalRes       = "Ваш результат: %d балл"
	leadboardTopTitle = "\nТоп-%d:\n"
	personalLeadData  = "@%s — %d балл"
	emptyRes          = "На квизе не присутствовало ни одного участника."

	authorMsgAthterRun  = "Квиз %q, с ID: %s, прошел успешно!\n\n"
	resultsCSVDocName   = "Leaderboard_ID%s_%q.csv"
	errorExportCSV      = "При попытке создания CSV файла произошла ошибка: %v."
	successfulExportCSV = "Файл с таблицей отправлен!"

	unsuccessfulRestartQuiz    = "При попытке перезапуска квиза произошла ошибка: %v"
	successfulRestartRunAnswer = "Квиз перезапущен!"
	successfulRestartRun       = "Квиз %q перезапущен!\nID запуска: %s\n\nСсылка для участников:\n%s\n\nКоличество участников: %d\n"

	emptyOwnerQuizzes      = "Вы ещё не загружали ни одного квиза. Пришлите мне JSON квиза или использую ссылку чтобы присоединиться к квизу."
	quizzesListTitle       = "Список загруженных Вами квизов:\n\n"
	quizzesListDescription = "%d. %q\nID: %s\nСсылка для получения JSON:\nhttps://t.me/%s?start=get_quiz_%s\n\n"
	emptyQuizID            = "Вы вызвали \"/start get_quiz\", но не передали ID. Вызовите \"/start get_quiz_QUIZID\", где QUIZID - корректный ID квиза."
	underfindQuizID        = "Вы вызвали \"/start get_quiz\", но передали несуществующий ID. Вызовите \"/start get_quiz_QUIZID\", где QUIZID - корректный ID квиза."
	quizJSONFileName       = "Quiz_%s_ID%s.json"

	quizRunsTitle               = "Список запускаемых Вами квизов:\n"
	quizRunsListTitle           = "\n%d. Запуски %q\n(ID: %s):\n\n"
	emptyQuizRunsList           = "\t\tВы не запускали этот квиз.\n\n\n"
	underfindQuizRunDescription = "\t\t%d.%d. Информация о запуске утрена\n\n"
	quizRunDescription          = "\t\t%d.%d. ID: %s\n\t\tКвиз не проведен до конца.\n\n"
	finishedQuizRunDescription  = "\t\t%d.%d. ID: %s\n\t\tСcылка для получения результатов в CSV:\n\t\thttps://t.me/%s?start=get_runres_%s\n\n"
	emptyRunResID               = "Вы вызвали \"/start get_runres\", но не передали ID. Вызовите \"/start get_runres_RUNID\", где RUNID - корректный ID квиза."
	underfindRunResID           = "Вы вызвали \"/start get_runres\", но передали несуществующий ID. Вызовите \"/start get_runres_RUNID\", где RUNID - корректный ID квиза."
)
