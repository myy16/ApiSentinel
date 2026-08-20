import { db } from "../../lib/db.js";
import { projects } from "../../db/schema/users.js";
import { eq, and } from "drizzle-orm";
import { CreateProjectInput } from "@apisentinel/shared";

export class ProjectService {
  static async listProjects(organizationId: string) {
    return db.query.projects.findMany({
      where: eq(projects.organizationId, organizationId),
      orderBy: (projects, { desc }) => [desc(projects.createdAt)],
    });
  }

  static async createProject(organizationId: string, input: CreateProjectInput) {
    const [project] = await db
      .insert(projects)
      .values({
        organizationId,
        name: input.name,
      })
      .returning();

    return project;
  }

  static async getProjectById(organizationId: string, projectId: string) {
    const project = await db.query.projects.findFirst({
      where: and(eq(projects.id, projectId), eq(projects.organizationId, organizationId)),
    });

    if (!project) {
      const error = new Error("Proje bulunamadı");
      (error as any).statusCode = 404;
      (error as any).code = "PROJECT_NOT_FOUND";
      throw error;
    }

    return project;
  }

  static async updateProject(organizationId: string, projectId: string, input: Partial<CreateProjectInput>) {
    await this.getProjectById(organizationId, projectId);

    const [updated] = await db
      .update(projects)
      .set({
        ...(input.name ? { name: input.name } : {}),
      })
      .where(and(eq(projects.id, projectId), eq(projects.organizationId, organizationId)))
      .returning();

    return updated;
  }

  static async deleteProject(organizationId: string, projectId: string) {
    await this.getProjectById(organizationId, projectId);

    await db
      .delete(projects)
      .where(and(eq(projects.id, projectId), eq(projects.organizationId, organizationId)));

    return { success: true };
  }
}
