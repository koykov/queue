# Queue

Queue это оболочка над Go каналами, которая имеет свойства:
* балансируемая
* с протечкой
* с повторной обработкой
* с Backoff механизмом
* планируемая
* с отложенным исполнением
* с учётом дедалйнов
* приоретизуемая
* с механизмом метрик
* с логированием

Очередь была реализована в ответ на необходимость создания множества однотипных очередей, с одинаковыми свойствами.
Из раза в раз создавать одинаковые каналы с разными обработчиками, покрывать их метриками и т.д. было слишком скучно и
в результате появилось это решение.

Queue позволяет абстрагироваться от деталей реализации самой очереди и сосредоточиться только на механизме обработки
элементов очереди. Достаточно написать реализацию обработчика, который реализует интерфейс
[Worker](https://github.com/koykov/queue/blob/master/interface.go#L18), привязать его к
[конфигу](https://github.com/koykov/queue/blob/master/config.go#L22) очереди и она сама выполнит всю работу.

Queue не является классической очередью с двумя методами `Enqueue` и `Dequeue`. У этой реализации отсутствует метод
`Dequeue` и вместо него предлагается обработчик `Worker`. Очередь сама выполняет `dequeue` операцию и отправляет
полученный элемент одному из активных воркеров. Таким образом, можно положить элемент в очередь посредством вызова
метода [`Enqueue`](https://github.com/koykov/queue/blob/master/interface.go#L6), а далее очередь сама проверит есть ли
доступный активный воркер, выполнит `dequeue` и отправит полученный элемент воркеру на обработку. Такая реализация
очереди похожа на шаблон [`Thread Pool`](https://en.wikipedia.org/wiki/Thread_pool), но возможности `Queue` выходят за
рамки этого шаблона.

Очередь настраивается посредством заполнения структуры
[`Config`](https://github.com/koykov/queue/blob/master/config.go#L22). Двумя обязательными параметрами конфига являются
`Capacity` и `Worker`.

Параметр `Capacity` задаёт размер (ёмкость) очереди в элементах. В действительности, этот параметр может быть опущен,
если заполнен параметр `QoS`, см раздел "Приоретизация".

Параметр `Worker` задаёт обработчик элементов очереди. Он должен реализовывать интерфейс
[`Worker`](https://github.com/koykov/queue/blob/master/interface.go#L18). Воркер может только обработать элемент очереди и сообщить успешно он обработан или нет. Есть
вспомогательный параметр `Workers`, который задаёт количество воркеров, но он не является обязательным, т.к. настройка
балансировки позволяет очереди игнорировать этот параметр. Но если вы не хотите балансируемую очередь, то следует
использовать этот параметр. Тогда получится классический `Thread Pool` со статическим количеством воркеров.

Под воркером подразумевается горутина (или возможность её запустить по необходимости, см. раздел "Балансировка"), внутри
которой работает объект обработчика.

В результате заполнения этих двух полей мы получим рабочую очередь с фиксированным размером, которая способна обработать
поступающие элементы.

Далее мы рассмотрим как включается и работает каждое из свойств очереди.

## Балансировка

Все очереди, с которыми я имел дело, имели переменную нагрузку. Днём она максимальная, а ночью сильно снижается.
Приходилось всё время держать включёнными максимальное количество воркеров, хотя ночью было достаточно одного-двух
процентов от этого количества. В Go это не является проблемой, т.к. горутины весьма дёшевы, но решить задачу
балансировки количества воркеров в зависимости от нагрузки было слишком заманчивой идеей и я не нашёл причин не решить
её.

Балансировка включается посредством задания параметров `WorkersMin` и `WorkersMax`, причём `WorkersMin` < `WorkersMax`.
Эти параметры не единственные, которые влияют на балансировку, но самые важные и их, как правило, достаточно. Замечу,
что включение этих параметров отключает параметр `Workers`.

`WorkersMin` задаёт количество перманентно активных воркеров. Вне зависимости от окружающих условий, количество активных
воркеров никогда не будет ниже этого значения.

`WorkersMax` ограничивает максимальное количество активных воркеров. Вне зависимости от условий работы, очередь не
сможет запустить больше чем `WorkersMax` воркеров. Подробнее что будет происходить в этом случае описано в разделе
"Протечка".

Все воркеры в промежутке между `WorkersMin` и `WorkersMax` могут находиться в трёх состояниях:
* _активный_ - воркер работает и обрабатывает элементы очереди.
* _спящий_ - воркер рабочий, но ничего не обрабатывает и ждёт, т.к. в очереди нет достаточного количества элементов.
Это состояние воркера не является постоянным. Если в течении какого-то времени не появится доступных для обработки
элементов, то воркер перейдёт в неактивное состояние.
* _неактивный_ - воркер (горутина) завершает работу. При необходимости очередь запустит новую горутину, в которой воркер 
займётся обработкой.

Очередь понимает, что пора запустить новый воркер в случае если
[рейт](https://github.com/koykov/queue/blob/master/interface.go#L12) очереди превышает параметр
`WakeupFactor` [0..0.999999].
Можно взять к примеру очередь с размером 100 и `WakeupFactor` 0.5. В случае если в ней накопится больше 50-и элементов,
будет запущен новый воркер. Если и его будет недостаточно, то будет запущен ещё один. Таким образом очередь будет
запускать по одному воркеру пока рейт не станет меньше 0.5 (или не будет достигнут `WorkersMax`, см раздел "Протечка").

Далее предположим, что нагрузка на очередь спала и количество активных воркеров стало избыточным. В этом случае очередь
смотрит на параметр `SleepFactor` [0.0.999999] (причём `SleepFactor` < `WakeupFactor`). Если рейт очереди стал меньше
чем `SleepFactor`, то очередь переведёт один из активных воркеров в спящее состояние. Затем ещё один, если условие 
рейт < `SleepFactor` продолжает выполняться. Так будет продолжаться пока рейт не превысит `SleepFactor` или не будет
достигнут `WorkersMin`. Спящий воркер не будет находиться в таком состоянии вечно. Есть параметр (промежуток времени)
`SleepInterval`, спустя который, если не повились элементы для обработки, воркер будет переведён в неактивное состояние
(горутина будет завершена). Спящее состояние необходимо для очередей, где нагрузка очень часто меняется. В таком случае
накладные расходы на создание/завершение горутин становится чувствительными. `SleepInterval` позволяет амортизировать
эту нагрузку.

Очередь постоянно балансирует количество воркеров таким образом, чтобы рейт находился в промежутке между `SleepFactor` и
`WakeupFactor`.

## Протечка

Представим себе очередь, нагрузка на которую выросла так сильно, что даже `WorkersMax` активных воркеров не справляются
с обработкой. В таких случаях блокируются все потоки, которые пытаются выполнить `Enqueue`.

Для решения этой проблемы был введён параметр `DLQ` (dead letter queue). Это вспомогательная очередь, реализующая
интерфейс [Enqueuer](https://github.com/koykov/queue/blob/master/interface.go#L4). То есть можно перенаправить излишек
входящих элементов в другую очередь или даже организовать цепочку очередей.

Установка этого параметра включает свойство "протечка" очереди. Это т.н.
["алгоритм текущего ведра"](https://en.wikipedia.org/wiki/Leaky_bucket). В терминах Go он называется leaky buffer и
описан в статье [Effective Go](https://golang.org/doc/effective_go#leaky_buffer).

Из коробоки доступна [DLQ-пустышка](https://github.com/koykov/queue/blob/master/dummy.go#L23), которая сольёт в мусор
все протёкшие элементы. Да, вы потеряете какое-то количество элементов, но сохраните своё приложение живым. Впрочем, уже
сейчас есть решение [dlqdump](https://github.com/koykov/dlqdump), которое позволит дампить протёкшие элементы в какое-то
хранилище, например на диск. Подробнее смотрите в описании к модулю.

В заключение к протечке, есть ещё параметр-флаг `FailToDLQ`. Если воркер сообщит очереди, что обработка элемента
провалилась, то очередь может направить элемент в `DLQ`, даже если загрузка очереди далека от протечки. Это может быть
полезным для случаев когда, например, провалилась отправка элемента на какой-то сетевой ресурс и имеет смысл попробовать
сделать это иным способом.

## Повторная обработка

Может так случиться, что одной попытки обработки элемента будет недостаточно. Например, провалится отправка HTTP запроса
в воркере и имеет смысл попробовать ещё раз. Есть параметр `MaxRetries`, который укажет сколько повторных(!) попыток
может предпринять воркер. Замечу, что в реальности попыток будет `MaxRetries` + 1, т.к. очередь считает, что самая
первая попытка обработки не должна считаться повтором. Все последующие попытки уже будут трактоваться как повторные и
лимитироваться `MaxRetries` параметром.

Это свойство может работать совместно с `FailToDQL` параметром. Т.е. после провала всех попыток обработки, элемент может
быть направлен в `DLQ` очередь.

## Backoff

Продолжение предыдущего пункта.

В случае если воркеры начнут сыпать ошибкми и потребуются массовые ретраи, очередь может не выдержать нагрузки в виде
как поступающих элементов, так и требующих попыток повторной обработки. Для этого случая добавлены два параметра:

* `RetryInterval` - задержка между попытками обработки
* `Backoff` - специальный компонент, который на основе `RetryInterval` высчитывает реальное время задержки

Из коробки доступны backoff:

* [Линейный](backoff/linear.go) (1, 2, 3, 4, ...)
* [Экспоненциальный](backoff/exponential.go) (1, 2, 4, 8, ...)
* [Квадратичный](backoff/quadratic.go) (1, 4, 9, 16, ...)
* [Полиномиальный](backoff/polynomial.go) (1, 8, 27, ...)
* [Логарифмический](backoff/logarithmic.go) (для задержки в 10: 7, 11, 14, ...)
* [Фибоначчи](backoff/fibonacci.go) (1, 1, 2, 3, 5, 8, ...)
* [Random](backoff/random.go) (`RetryInterval/2 + random(RetryInterval)`)

## Планирование нагрузки

Предположим, что вам известна периодичность роста/спада нагрузки на очередь. Например, с 8:00 до 12:00 и с 04:00
до 18:00 нагрузка средняя и хватает 5-и воркеров. С 12:00 до 16:00 нагрузка максимальна и требует как минимум
10 активных воркеров. А ночью нагрузка очень сильно спадает и достаточно только одного воркера. Для таких случаев был
разработан [планировщик](https://github.com/koykov/queue/blob/master/schedule.go) параметров очереди. Он позволяет
менять некоторые параметры очереди для определённых промежутков времени:
* `WorkersMin`
* `WorkersMax`
* `WakeupFactor`
* `SleepFactor`

Указанные параметры перезаписывают в заданном промежутке времени аналогичные параметры из конфига.

Для приведённого выше примера, инициализация планировщика будет выглядеть вот так:
```go
sched := NewSchedule()
sched.AddRange("08:00-12:00", ScheduleParams{WorkersMin: 5, WorkersMax: 10})
sched.AddRange("12:00-16:00", ScheduleParams{WorkersMin: 10, WorkersMax: 20})
sched.AddRange("16:00-18:00", ScheduleParams{WorkersMin: 5, WorkersMax: 10})
config := Config{
	...
	WorkersMin: 1,
	WorkersMax: 4,
	Schedule: sched,
	...
}
```
Такой конфиг будет для следующих промежутков времени балансировать очередь:
* между 5 и 10 воркерами в промежутке 8:00 - 12:00
* между 10 и 20 воркерами в промежутке 12:00 - 16:00
* между 5 и 10 воркерами в промежутке 16:00 - 18:00
* между 1 и 4 воркерами всё остальное время

Свойство было разработано с целью упростить балансировку в часы пик. В принципе, для решения этой проблемы достаточно
указать достаточно большой `SleepInterval`. Но лучше иметь несколько вариантов, чем один.

## Отложенное исполнение

Однажды возникла необходимость обрабатывать элементы очереди не сразу, а спустя определённый промежуток времени после
добавления в очередь. Специально для этой цели был добавлен параметр-интервал `DelayInterval`. Установка этого параметра
включает отложенное исполнение и гарантирует, что элемент будет принят в обработку воркером спустя как минимум
`DelayInterval` промежуток времени.

Этот параметр по смыслу противоположен параметру `DeadlineInterval`.

## Учёт дедлайнов

У высоконагруженных очередей может возникнуть ситуация, когда элемент доходит до воркера спустя слишком большой
промежуток времени и его обработка теряет смысл. Для этого случая предусмотрен параметр `DeadlineInterval`, который
отсчитывается с момента попадания элемента в очередь и учитывается перел отправкой в воркер. Если срок вышел, то элемент
не будет взят в обработку.

Этот араметр может работать совместно с флагом `DeadlineToDQL`.

Этот параметр по смыслу противоположен параметру `DelayInterval`.

## Приоретизация

По умолчанию очередь работает по FIFO алгоритму. Это прекрасно работает до момента, когда в очередь начинают поступать
конвергентные данные. Если у вас есть очередь, куда одновременно поступают элементы как требующие срочной обработки,
так и не срочные, то рано или поздно возникнет ситуация, когда в начале очереди будут неприоритетные элементы, а в конце
приоритетные. К моменту пока очередь доберётся до приоритетных элементов они уже устареют и их обработка потеряет всякий
смысл. В компьютерных сетях эта проблема была решена уже давно и это решение называется
[Quality of Service (QoS)](https://en.wikipedia.org/wiki/Quality_of_service).

В конфиге очереди есть параметр `QoS`, который является структурой [`qos.Config`](qos/config.go). Заполнение этого
праметра сделает очередь приоретизируемой. Детали настройки см. в [readme](qos/readme.ru.md).

## Покрытие метриками

`Queue` имеет параметр `MetricsWriter` где можно задать объект, реализующий интерфейс
[`MetricsWriter`](https://github.com/koykov/queue/blob/master/metrics.go#L7). Описание всех методов см. в исходном коде
интерфейса, он достаточно простой.

На данный момент написаны две реализации этого интерфейса:
* [`log.MetricsWriter`](metrics/log)
* [`prometheus.MetricsWriter`](metrics/prometheus)

Первый бессмысленно использовать в продакшн условиях и он создавался для упрощения отладки при разработке. А вот Prometheus
версия полностью рабочая и протестированная. Аналогичным образом, можно написать реализацию `MetricsWriter`-а для иной
TSDB, которую вы используете.

## Коробочные воркеры

`queue` по умолчанию предлагает три воркера:
* [transit](https://github.com/koykov/queue/blob/master/worker/transit.go) просто перенаправляет элемент в другую
очередь.
* [chain](https://github.com/koykov/queue/blob/master/worker/chain.go) объединяет несколько воркеров внутри одного.
Элемент будет синхронно обработан каждым из дочерних воркеров. Можно построить цепочку из воркеров-обработчиков и
завершить её `transit`-воркером.
* [async_chain](https://github.com/koykov/queue/blob/master/worker/async_chain.go) также объединяет несколько воркеров
внутри одного, но элемент обрабатывается дочерними воркерами асинхронно.

Используя эти воркеры можно построить сколь угодно сложное дерево обработки элемента и в конце отправить его в следущую
очередь.

## Логирование

`Queue` умеет логировать свои внутренние процессы с помощью параметра `Logger`, куда можно задать логгер, реализующий
интерфейс [`Logger`](https://github.com/koykov/queue/blob/master/logger.go). Он очень простой и пригодится только для
отладочных и/или демонстрационных целей.

## Demo сцена

В процессе разработки одной из основных трудностей стала невозможность написания тестов. Эта проблема была решена через
написание специального [демонстрационного проекта](https://github.com/koykov/demo/tree/master/queue), где обкатывались
всевозможные сценарии работы очереди, комбинации свойств/параметров и любых идей, что приходили в голову. Это готовый
проект, под него написан `Docker` контейнер, включащий `Grafana`, `Prometheus` и `queue daemon`. Управляется посредством
HTTP запросов, они описаны в [readme](https://github.com/koykov/demo/blob/master/queue/readme.md).

Типовые сценарии хранятся [тут](https://github.com/koykov/demo/tree/master/queue/request).

## Ссылки

* https://en.wikipedia.org/wiki/Thread_pool
* https://en.wikipedia.org/wiki/Leaky_bucket
* https://golang.org/doc/effective_go#leaky_buffer
* https://en.wikipedia.org/wiki/Dead_letter_queue
* https://cloud.yandex.com/en/docs/message-queue/concepts/dlq
* https://en.wikipedia.org/wiki/Quality_of_service
