import os
import cv2
import mediapipe as mp
import pandas as pd
import numpy as np


# Mediapipe FaceMesh
mp_face_mesh = mp.solutions.face_mesh

# вырез региона по landmark-точкам
def crop_region(img, landmarks, indices, expand=5):
    h, w, _ = img.shape
    points = [(int(landmarks[i].x * w), int(landmarks[i].y * h)) for i in indices]
    x_min = max(min([p[0] for p in points]) - expand, 0)
    x_max = min(max([p[0] for p in points]) + expand, w)
    y_min = max(min([p[1] for p in points]) - expand, 0)
    y_max = min(max([p[1] for p in points]) + expand, h)
    return img[y_min:y_max, x_min:x_max]


# вычисление yaw, pitch, roll через solvePnP
def get_head_pose(landmarks, img_w, img_h):
    # 3D точки лица (условные)
    face_3d = np.array([
        (0.0, 0.0, 0.0),     # нос
        (0.0, -330.0, -65.0),# подбородок
        (-225.0, 170.0, -135.0), # левый угол глаза
        (225.0, 170.0, -135.0),  # правый угол глаза
        (-150.0, -150.0, -125.0),# левый угол рта
        (150.0, -150.0, -125.0), # правый угол рта
    ], dtype=np.float64)

    # 2D координаты из Mediapipe
    face_2d = np.array([
        (landmarks[1].x * img_w, landmarks[1].y * img_h),   # нос
        (landmarks[152].x * img_w, landmarks[152].y * img_h), # подбородок
        (landmarks[263].x * img_w, landmarks[263].y * img_h), # правый глаз угол
        (landmarks[33].x * img_w, landmarks[33].y * img_h),   # левый глаз угол
        (landmarks[287].x * img_w, landmarks[287].y * img_h), # правый рот
        (landmarks[57].x * img_w, landmarks[57].y * img_h),   # левый рот
    ], dtype=np.float64)

    # камера параметры
    focal_length = img_w
    cam_matrix = np.array([
        [focal_length, 0, img_w / 2],
        [0, focal_length, img_h / 2],
        [0, 0, 1]
    ])
    dist_matrix = np.zeros((4, 1), dtype=np.float64)

    # SolvePnP
    success, rot_vec, trans_vec = cv2.solvePnP(face_3d, face_2d, cam_matrix, dist_matrix)
    rmat, _ = cv2.Rodrigues(rot_vec)
    angles, _, _, _, _, _ = cv2.RQDecomp3x3(rmat)

    pitch, yaw, roll = [a * 180 for a in angles]  # в градусах
    return pitch, yaw, roll


def process_dataset(input_dir, output_dir):
    os.makedirs(output_dir, exist_ok=True)

    # индексы Mediapipe для глаз/рта
    LEFT_EYE = list(range(362, 382)) # левый глаз
    RIGHT_EYE = list(range(133, 154)) # правый глаз
    MOUTH = list(range(78, 95)) # рот

    with mp_face_mesh.FaceMesh(static_image_mode=True, max_num_faces=1, refine_landmarks=True) as face_mesh:
        for cls in ["Drowsy", "Non Drowsy"]:
            cls_in = os.path.join(input_dir, cls)
            cls_out = os.path.join(output_dir, cls)
            os.makedirs(os.path.join(cls_out, "eyes"), exist_ok=True)
            os.makedirs(os.path.join(cls_out, "mouths"), exist_ok=True)

            pose_data = []

            for fname in os.listdir(cls_in):
                if not fname.lower().endswith((".jpg", ".png", ".jpeg")):
                    continue

                path_in = os.path.join(cls_in, fname)
                img = cv2.imread(path_in)
                if img is None:
                    continue

                h, w, _ = img.shape
                rgb = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)
                results = face_mesh.process(rgb)

                if not results.multi_face_landmarks:
                    print(f"[!] Лицо не найдено: {path_in}")
                    continue

                landmarks = results.multi_face_landmarks[0].landmark

                # вырез глаз
                left_eye = crop_region(img, landmarks, LEFT_EYE)
                right_eye = crop_region(img, landmarks, RIGHT_EYE)
                eyes = cv2.hconcat([left_eye, right_eye])

                # вырез рта
                mouth = crop_region(img, landmarks, MOUTH)

                # сохранение
                eye_path = os.path.join(cls_out, "eyes", fname)
                mouth_path = os.path.join(cls_out, "mouths", fname)
                cv2.imwrite(eye_path, eyes)
                cv2.imwrite(mouth_path, mouth)

                # поза головы
                pitch, yaw, roll = get_head_pose(landmarks, w, h)
                pose_data.append({
                    "file": fname,
                    "class": cls,
                    "pitch": pitch,
                    "yaw": yaw,
                    "roll": roll
                })

                # сохранение CSV
            df = pd.DataFrame(pose_data)
            df.to_csv(os.path.join(cls_out, "poses.csv"), index=False)
            print(f"класс {cls} обработан, сохранено {len(pose_data)} записей.")

if __name__ == "__main__":
    input_dir = "AI_DETECTOR/archive/Driver Drowsiness Dataset (DDD)"
    output_dir = "processed_dataset"
    process_dataset(input_dir, output_dir)
    print("обработка завершена! Данные сохранены в processed_dataset/")