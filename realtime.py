import os, time, cv2, torch
import numpy as np
from PIL import Image
import torchvision.transforms as T

from network import TinyCNN

WEIGHTS = "checkpoints/final_simple.pth"  # веса из training_simple.py
IMG_SIZE = 224
THRESH = 0.5
CAM_INDEX = 0
OVERLAY_ALPHA = 0.45
ALARM_TEXT = "DROWSY — WAKE UP!"
BLINK_PERIOD = 1.0

to_tensor = T.Compose([
    T.Resize((IMG_SIZE, IMG_SIZE)),
    T.ToTensor(),
    T.Normalize([0.5,0.5,0.5],[0.5,0.5,0.5])
])

device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
model = TinyCNN(num_classes=1).to(device).eval()
if not os.path.isfile(WEIGHTS):
    raise FileNotFoundError(WEIGHTS)
state = torch.load(WEIGHTS, map_location=device)
model.load_state_dict(state)
print("Loaded:", WEIGHTS, "| Device:", device)

# детектор лица (Haar)
face_cascade = cv2.CascadeClassifier(cv2.data.haarcascades + "haarcascade_frontalface_default.xml")

cap = cv2.VideoCapture(CAM_INDEX)
if not cap.isOpened():
    raise RuntimeError("Camera not opened")

last_time = time.time()
while True:
    ret, frame = cap.read()
    if not ret: break
    h, w = frame.shape[:2]

    # ищем лицо
    gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
    faces = face_cascade.detectMultiScale(gray, scaleFactor=1.2, minNeighbors=5, minSize=(80,80))
    if len(faces) > 0:
        x,y,fw,fh = max(faces, key=lambda f: f[2]*f[3])  # самое большое лицо
        x2, y2 = x+fw, y+fh
        cv2.rectangle(frame, (x,y), (x2,y2), (0,255,255), 2)
        crop = frame[y:y2, x:x2]
    else:
        crop = frame

    pil = Image.fromarray(cv2.cvtColor(crop, cv2.COLOR_BGR2RGB))
    x = to_tensor(pil).unsqueeze(0).to(device)

    with torch.no_grad():
        logit = model(x)
        prob = torch.sigmoid(logit).item()

    status = f"{'Drowsy' if prob>=THRESH else 'Alert'} {prob:.2f}"

    # FPS
    now = time.time()
    fps = 1.0/(now-last_time) if now>last_time else 0.0
    last_time = now

    color = (0,0,255) if prob>=THRESH else (0,255,0)
    cv2.putText(frame, status, (20,40), cv2.FONT_HERSHEY_SIMPLEX, 1.0, color, 2)
    cv2.putText(frame, f"FPS: {fps:.1f}", (20,75), cv2.FONT_HERSHEY_SIMPLEX, 0.8, (255,255,255), 2)

    # яркая плашка при сонливости
    if prob >= THRESH:
        overlay = frame.copy()
        overlay[:] = (0,0,255)
        cv2.addWeighted(overlay, OVERLAY_ALPHA, frame, 1-OVERLAY_ALPHA, 0, frame)
        if (time.time() % BLINK_PERIOD) < (BLINK_PERIOD/2):
            (tw, th), _ = cv2.getTextSize(ALARM_TEXT, cv2.FONT_HERSHEY_DUPLEX, 1.2, 3)
            cv2.putText(frame, ALARM_TEXT, ((w-tw)//2, h//2), cv2.FONT_HERSHEY_DUPLEX, 1.2, (255,255,255), 3)

    cv2.imshow("Drowsiness Simple (q to quit)", frame)
    k = cv2.waitKey(1) & 0xFF
    if k == ord('q'): break

cap.release()
cv2.destroyAllWindows()