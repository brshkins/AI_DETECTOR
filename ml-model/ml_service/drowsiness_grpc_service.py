import os
import sys
import cv2
import torch
import numpy as np
from PIL import Image
import torchvision.transforms as T
from concurrent import futures
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from network import TinyCNN
import grpc
import time
import signal
import threading
from drowsiness_pb2 import VideoFrame, DetectionResult, HealthStatus, Empty
from drowsiness_pb2_grpc import DrowsinessDetectionServicer, add_DrowsinessDetectionServicer_to_server

shutdown_event = threading.Event()
def signal_handler(sig, frame):
    print('\n[SHUTDOWN] Получен сигнал остановки...')
    shutdown_event.set()
    sys.exit(0)

# Регистрируем обработчик сигналов
signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)


WEIGHTS_PATH = os.path.join(os.path.dirname(__file__), "checkpoints", "final_simple.pth")
IMG_SIZE = 224
THRESHOLD = 0.5
GRPC_PORT = 9000

# ИНИЦИАЛИЗАЦИЯ МОДЕЛИ
print("Initializing model...")

device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
print(f"Device: {device}")

# Загружаем модель один раз
model = TinyCNN(num_classes=1).to(device).eval()

if not os.path.isfile(WEIGHTS_PATH):
    raise FileNotFoundError(f"Model weights not found: {WEIGHTS_PATH}")

state = torch.load(WEIGHTS_PATH, map_location=device)
model.load_state_dict(state)
print(f"Model loaded: {WEIGHTS_PATH}")

# Трансформация изображения
to_tensor = T.Compose([
    T.Resize((IMG_SIZE, IMG_SIZE)),
    T.ToTensor(),
    T.Normalize([0.5, 0.5, 0.5], [0.5, 0.5, 0.5])
])

# Детектор лица (Haar Cascade)
face_cascade = cv2.CascadeClassifier(
    cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
)


class DrowsinessDetectionService(DrowsinessDetectionServicer):
    #gRPC Service для обнаружения сонливости

    def DetectDrowsiness(self, request, context):
        import time
        start_time = time.time()

        try:
            # Декодируем изображение
            nparr = np.frombuffer(request.frame_data, np.uint8)
            frame = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

            if frame is None:
                return DetectionResult(
                    frame_id=request.frame_id,
                    drowsy_score=0.0,
                    is_drowsy=False,
                    error="Failed to decode image"
                )

            # Детектируем лицо
            gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
            faces = face_cascade.detectMultiScale(
                gray,
                scaleFactor=1.2,
                minNeighbors=5,
                minSize=(80, 80)
            )

            # Берём самое большое лицо
            if len(faces) > 0:
                x, y, fw, fh = max(faces, key=lambda f: f * f)
                crop = frame[y:y + fh, x:x + fw]
            else:
                crop = frame

            # Преобразуем в тензор
            pil_img = Image.fromarray(cv2.cvtColor(crop, cv2.COLOR_BGR2RGB))
            tensor_img = to_tensor(pil_img).unsqueeze(0).to(device)

            # Запускаем модель
            with torch.no_grad():
                logit = model(tensor_img)
                score = torch.sigmoid(logit).item()

            # Вычисляем время обработки
            processing_time = int((time.time() - start_time) * 1000)

            return DetectionResult(
                sequence_number=request.sequency_number,
                drowsiness_score=float(score),
                is_drowsy=bool(score >= THRESHOLD),
                inference_time_ms=processing_time,
                timestamp=int(time.time() * 1000)
            )

        except Exception as e:
            print(f"Error in DetectDrowsiness: {e}")
            return DetectionResult(
                sequence_number=request.sequence_number,
                drowsiness_score=0.0,
                is_drowsy=False,
                error=str(e)
            )

    def Health(self, request, context):
        # Проверка здоровья сервиса
        return HealthStatus(
            status="healthy",
            model_loaded=True,
            device=str(device)
        )

def DetectDrowsinessStream(self, request_iterator, context):
    """Потоковая обработка видео кадров"""
    for request in request_iterator:
        try:
            # Используем тот же код что и в DetectDrowsiness
            nparr = np.frombuffer(request.frame_data, np.uint8)
            frame = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

            if frame is None:
                continue

            # Детектируем лицо
            gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
            faces = face_cascade.detectMultiScale(
                gray,
                scaleFactor=1.2,
                minNeighbors=5,
                minSize=(80, 80)
            )

            if len(faces) > 0:
                x, y, fw, fh = max(faces, key=lambda f: f * f)
                crop = frame[y:y + fh, x:x + fw]
            else:
                crop = frame

            # Преобразуем в тензор
            pil_img = Image.fromarray(cv2.cvtColor(crop, cv2.COLOR_BGR2RGB))
            tensor_img = to_tensor(pil_img).unsqueeze(0).to(device)

            # Запускаем модель
            with torch.no_grad():
                logit = model(tensor_img)
                score = torch.sigmoid(logit).item()

            # Отправляем результат
            yield DetectionResult(
                sequence_number=request.sequency_number,
                drowsiness_score=float(score),
                is_drowsy=bool(score >= THRESHOLD),
                timestamp=int(time.time() * 1000)
            )

        except Exception as e:
            print(f"Stream error: {e}")
            continue


def serve():
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ('grpc.max_receive_message_length', 50 * 1024 * 1024),
            ('grpc.max_send_message_length', 50 * 1024 * 1024),
        ]
    )
    add_DrowsinessDetectionServicer_to_server(
        DrowsinessDetectionServicer(), server
    )

    port = '9000'
    server.add_insecure_port(f'[::]:{port}')
    server.start()

    print(f'[gRPC] Сервер запущен на порту {port}')
    print('[INFO] Нажмите Ctrl+C для остановки...\n')

    try:
        # ✅ ВМЕСТО while True:
        shutdown_event.wait()  # Ждём сигнала остановки
    except KeyboardInterrupt:
        print('\n[SHUTDOWN] Остановка сервера...')
    finally:
        server.stop(grace=5)  # Graceful shutdown с таймаутом 5 секунд
        print('[SHUTDOWN] Сервер остановлен.')
        sys.exit(0)

if __name__ == "__main__":
    serve()